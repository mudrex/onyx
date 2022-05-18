package audit

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	s3Lib "github.com/aws/aws-sdk-go-v2/service/s3"
	configPkg "github.com/mudrex/onyx/pkg/config"
	"github.com/mudrex/onyx/pkg/logger"
)

func Log(ctx context.Context, message string) {
	f, err := os.OpenFile(configPkg.Config.LocalLogFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		logger.Error("Unable to open file. Error: %s", err.Error())
		return
	}

	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		logger.Error("Unable to get file size. Error: %s", err.Error())
		return
	}

	if fi.Size() > 1000000 { // greater than 1mb
		fmt.Println("Flushing log file. Might take some time for the next action.")

		file, _ := os.Open(configPkg.Config.LocalLogFilename)
		done := uploadToS3(ctx, file)
		file.Close()
		if done {
			f.Truncate(0)
		}
	}

	if _, err = f.WriteString(fmt.Sprintf("%s %s\n", time.Now().Format(time.RFC3339), message)); err != nil {
		logger.Error("Unable to write to log file. Error: %s", err.Error())
		return
	}
}

func uploadToS3(ctx context.Context, body io.Reader) bool {
	if len(configPkg.Config.AuditBucket) == 0 {
		logger.Error("Unable to flush logs to s3. Please contact platform team.")
		return false
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(configPkg.GetRegion()))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	s3Handler := s3Lib.NewFromConfig(cfg)

	now := time.Now()

	uploader := manager.NewUploader(s3Handler)
	_, err = uploader.Upload(ctx, &s3Lib.PutObjectInput{
		Bucket: aws.String(configPkg.Config.AuditBucket),
		Key: aws.String(
			fmt.Sprintf(
				"onyx/logs/%s/dt=%s/hour=%s/archive_log-%s",
				configPkg.Config.Environment,
				now.Format("20060102"),
				now.Format("15"),
				now.Format("04:05"),
			),
		),
		Body: body,
	})

	return err == nil
}
