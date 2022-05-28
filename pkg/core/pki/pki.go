package pki

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/core/secretsmanager"
	"github.com/mudrex/onyx/pkg/logger"
	"github.com/mudrex/onyx/pkg/utils"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type CASecret struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
}

var caSecret = CASecret{}

var certificateSubject = map[string]string{
	"country":            strings.ToUpper(config.Config.CertificateSubject.Country),
	"province":           cases.Title(language.English).String(config.Config.CertificateSubject.Province),
	"locality":           cases.Title(language.English).String(config.Config.CertificateSubject.Locality),
	"organization":       cases.Title(language.English).String(config.Config.CertificateSubject.Organization),
	"organizationalUnit": cases.Title(language.English).String(config.Config.CertificateSubject.OrganizationalUnit),
}

func CreateCertificate(ctx context.Context, cfg aws.Config, name string, certType string, dnsNames []string, ipAddresses []string) error {
	return create(ctx, cfg, name, certType, dnsNames, ipAddresses)
}

func create(ctx context.Context, cfg aws.Config, name string, certType string, dnsNames []string, ipAddresses []string) error {
	// Clean input IPs
	ips, err := utils.GetIPsFromStrings(ipAddresses)
	if err != nil {
		return err
	}
	logger.Info("Received IP Addresses |  %v", ips)

	// Clean input DNSs
	dnsList, err := utils.GetDNSListFromStrings(dnsNames)
	if err != nil {
		return err
	}
	logger.Info("Received DNS Names |  %v", dnsList)

	//Determine name type and add to corresponding list
	nameArr := []string{name}
	logger.Info("%v", nameArr)
	nameIP, nameIPErr := utils.GetIPsFromStrings(nameArr)
	nameDNS, nameDNSErr := utils.GetDNSListFromStrings(nameArr)

	if nameIPErr == nil {
		logger.Info("Common Name is an IP, adding to IP SAN entries")
		ips = append(ips, nameIP...)
	}
	logger.Info("Final IP Addresses |  %v", ips)

	if nameDNSErr == nil {
		logger.Info("Common Name is a DNS, adding to DNS SAN entries")
		dnsList = append(dnsList, nameDNS...)
	}
	logger.Info("Final DNS Names |  %v", dnsList)

	if nameIPErr == nil && nameDNSErr == nil {
		logger.Error("Common Name is neither a DNS nor an IP, please check name %s", name)
		return errors.New("invalid common name")
	}

	// Get CA from Secrets Manager
	secretString := secretsmanager.GetSecret(ctx, cfg, config.Config.CASecretName)
	err = json.Unmarshal([]byte(secretString), &caSecret)
	if err != nil {
		return err
	}
	//Decode CA certificate
	caCertificateBlock, _ := pem.Decode([]byte(strings.Replace(caSecret.Certificate, `\n`, "\n", -1)))
	if caCertificateBlock == nil {
		return errors.New("unable to decode CA certificate")
	}
	caCertificate, err := x509.ParseCertificate(caCertificateBlock.Bytes)
	if err != nil {
		return err
	}

	//Decode CA private key
	caPrivateKeyBlock, _ := pem.Decode([]byte(strings.Replace(caSecret.PrivateKey, `\n`, "\n", -1)))
	caPrivateKey, err := x509.ParsePKCS1PrivateKey(caPrivateKeyBlock.Bytes)
	if err != nil {
		return err
	}
	logger.Info("Received CA for signing | %s", caCertificate.Subject.CommonName)

	// Get Certificate
	var certificate *x509.Certificate
	switch certType {
	case "server":
		certificate = getServerCertificate(ctx, cfg, name, ips, dnsList)
	case "client":
		certificate = getClientCertificate(ctx, cfg, name, ips, dnsList)
	default:
		return errors.New("unrecognized type, must be one of server|client")
	}

	// Create private key
	certPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	//Sign with CA
	certBytes, err := x509.CreateCertificate(rand.Reader, certificate, caCertificate, &certPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		return err
	}

	// Write certificate to file
	fileNamePrefix := name

	certFileName := fileNamePrefix + ".cert.pem"
	certFile, err := os.Create(certFileName)
	if err != nil {
		return err
	}
	err = pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return err
	}
	certFile.Close()

	// Write private key to file
	certPrivateKeyFileName := fileNamePrefix + ".key"
	pemPrivateFile, err := os.Create(certPrivateKeyFileName)
	if err != nil {
		return err
	}
	err = pem.Encode(pemPrivateFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivateKey),
	})
	if err != nil {
		return err
	}
	pemPrivateFile.Close()

	logger.Info("Successfully created private key | %s, and certificate | %s", certPrivateKeyFileName, certFileName)

	return nil
}

func getServerCertificate(ctx context.Context, cfg aws.Config, name string, ips []net.IP, dnsNames []string) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Country:            []string{certificateSubject["country"]},
			Province:           []string{certificateSubject["province"]},
			Locality:           []string{certificateSubject["locality"]},
			Organization:       []string{certificateSubject["organization"]},
			OrganizationalUnit: []string{certificateSubject["organizationalUnit"]},
			CommonName:         name,
		},
		IPAddresses:           ips,
		DNSNames:              dnsNames,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		BasicConstraintsValid: false,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		ExtraExtensions:       []pkix.Extension{},
	}
}

func getClientCertificate(ctx context.Context, cfg aws.Config, name string, ips []net.IP, dnsNames []string) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Country:            []string{certificateSubject["country"]},
			Province:           []string{certificateSubject["province"]},
			Locality:           []string{certificateSubject["locality"]},
			Organization:       []string{certificateSubject["organization"]},
			OrganizationalUnit: []string{certificateSubject["organizationalUnit"]},
			CommonName:         name,
		},
		IPAddresses:           ips,
		DNSNames:              dnsNames,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		BasicConstraintsValid: false,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
}

func getExtensions() []pkix.Extension {
	// TODO: Figure out marshalling
	basicConstraintsValue, _ := asn1.MarshalWithParams("CA:FALSE", `asn1:"printable"`)
	basicConstraints := pkix.Extension{
		Id:       asn1.ObjectIdentifier{2, 5, 29, 19},
		Critical: false,
		Value:    basicConstraintsValue,
	}

	netScapeCertTypeValue, err := asn1.MarshalWithParams([]string{"SSL Server"}, `asn1:"utf8"`)
	if err != nil {
		logger.Error("%v", err)
	}
	netScapeCertType := pkix.Extension{
		Id:       asn1.ObjectIdentifier{2, 16, 840, 1, 113730, 1, 1},
		Critical: false,
		Value:    netScapeCertTypeValue,
	}

	return []pkix.Extension{basicConstraints, netScapeCertType}
}
