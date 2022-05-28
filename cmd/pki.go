package cmd

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	configPkg "github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/core/pki"
	"github.com/spf13/cobra"
)

var certificateType string
var certificateDNSNames []string
var certificateIPAddresses []string

var pkiCommand = &cobra.Command{
	Use:   "pki",
	Short: "Actions to be performed on the PKI infrastructure",
}

var pkiCertificateCommand = &cobra.Command{
	Use:   "certificate",
	Short: "Creates and returns a certificate signed by a CA.",
}

var pkiCertificateCreateCommand = &cobra.Command{
	Use:   "create [name] {[--type type] | [--dns-names names] | [--ip-addresses ips]}",
	Short: "Creates a X.509 certificate for a server/client.",
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return configPkg.LoadConfig()
	},
	Example: `
	onyx pki certificate create *.client --type server
	onyx pki certificate create *.server --type server --dns-names *.*.test --ip-addresses 1.1.1.1
	onyx pki certificate create *.client --type client --dns-names *.*.test
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(configPkg.GetRegion()))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		if certificateType != "server" && certificateType != "client" {
			return errors.New("invalid type, must be one of server|client")
		}

		if args[0] != "" {
			return pki.CreateCertificate(ctx, cfg, args[0], certificateType, certificateDNSNames, certificateIPAddresses)
		}

		return errors.New("please specify a common name for the certificate")
	},
}

func init() {
	pkiCommand.AddCommand(pkiCertificateCommand)
	pkiCertificateCommand.AddCommand(pkiCertificateCreateCommand)

	pkiCertificateCreateCommand.Flags().StringVarP(&certificateType, "type", "t", "server", "End entity to issue the certificate for. Valid values are server|client")
	pkiCertificateCreateCommand.Flags().StringSliceVarP(&certificateDNSNames, "dns-names", "d", []string{}, "List of DNS Names to add in the SAN entry. By default, the common name will be a part of the SAN.")
	pkiCertificateCreateCommand.Flags().StringSliceVarP(&certificateIPAddresses, "ip-addresses", "i", []string{}, "List of IP Addresses to add in the SAN entry. By default, the common name (if it is an IP) will be a part of the SAN.")
}
