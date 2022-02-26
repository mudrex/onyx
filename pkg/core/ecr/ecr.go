package ecr

import (
	"context"
	"regexp"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecrLib "github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/mudrex/onyx/pkg/logger"
)

var rTag, _ = regexp.Compile(`v\d+.\d+.\d+`)
var stagingRTag, _ = regexp.Compile(`(staging|master).*`)
var sandboxRTag, _ = regexp.Compile(`sandbox.*`)

type Image struct {
	types.ImageIdentifier

	PushedAt int64
}

func getAllImages(ctx context.Context, ecrHandler *ecrLib.Client, repository string) ([]types.ImageDetail, error) {
	images := make([]types.ImageDetail, 0)
	var nextToken *string
	i := 0

	for nextToken != nil || i == 0 {
		o2, err := ecrHandler.DescribeImages(ctx, &ecrLib.DescribeImagesInput{
			RepositoryName: aws.String(repository),
			NextToken:      nextToken,
		})
		if err != nil {
			break
		}

		nextToken = o2.NextToken
		images = append(images, o2.ImageDetails...)
		i++
	}

	return images, nil
}

func getAllRepositories(ctx context.Context, ecrHandler *ecrLib.Client) ([]string, error) {
	repositories := make([]string, 0)

	output, err := ecrHandler.DescribeRepositories(ctx, &ecrLib.DescribeRepositoriesInput{})
	if err != nil {
		return repositories, err
	}

	for _, repository := range output.Repositories {
		repositories = append(repositories, aws.ToString(repository.RepositoryName))
	}

	return repositories, nil
}

func Cleanup(ctx context.Context, cfg aws.Config, repository string) error {
	ecrHandler := ecrLib.NewFromConfig(cfg)

	repositories := make([]string, 0)
	if repository == "" {
		allRepositories, err := getAllRepositories(ctx, ecrHandler)
		if err != nil {
			return err
		}

		repositories = allRepositories
	} else {
		repositories = append(repositories, repository)
	}

	for _, repository := range repositories {
		images, err := getAllImages(ctx, ecrHandler, repository)
		if err != nil {
			return err
		}

		stagingImages := make(map[int64]Image)
		sandboxImages := make(map[int64]Image)
		prodImages := make(map[int64]Image)
		for _, image := range images {
			for _, tag := range image.ImageTags {
				if stagingRTag.MatchString(tag) {
					stagingImages[image.ImagePushedAt.Unix()] = Image{
						ImageIdentifier: types.ImageIdentifier{
							ImageDigest: image.ImageDigest,
							ImageTag:    aws.String(tag),
						},
						PushedAt: image.ImagePushedAt.Unix(),
					}
				} else if rTag.MatchString(tag) {
					prodImages[image.ImagePushedAt.Unix()] = Image{
						ImageIdentifier: types.ImageIdentifier{
							ImageDigest: image.ImageDigest,
							ImageTag:    aws.String(tag),
						},
						PushedAt: image.ImagePushedAt.Unix(),
					}
				} else if sandboxRTag.MatchString(tag) {
					sandboxImages[image.ImagePushedAt.Unix()] = Image{
						ImageIdentifier: types.ImageIdentifier{
							ImageDigest: image.ImageDigest,
							ImageTag:    aws.String(tag),
						},
						PushedAt: image.ImagePushedAt.Unix(),
					}
				}
			}
		}

		if err := deleteImages(ctx, ecrHandler, repository, stagingImages, "staging"); err != nil {
			return err
		}

		if err := deleteImages(ctx, ecrHandler, repository, prodImages, "prod"); err != nil {
			return err
		}

		if err := deleteImages(ctx, ecrHandler, repository, sandboxImages, "sandbox"); err != nil {
			return err
		}
	}

	return nil
}

func deleteImages(
	ctx context.Context,
	ecrHandler *ecrLib.Client,
	repository string,
	images map[int64]Image,
	identifier string,
) error {
	keys := make([]int64, len(images))
	i := 0
	for k := range images {
		keys[i] = k
		i++
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	imagesToDelete := make([]types.ImageIdentifier, 0)
	j := len(keys) - 2
	for _, k := range keys {
		if j > 0 {
			imagesToDelete = append(imagesToDelete, images[int64(k)].ImageIdentifier)
			j--
		}
	}

	if len(imagesToDelete) == 0 {
		logger.Info("no images to delete %s for (%s)", repository, identifier)
		return nil
	}

	var imagesToDeleteChunks [][]types.ImageIdentifier

	chunkSize := 50

	for i := 0; i < len(imagesToDelete); i += chunkSize {
		end := i + chunkSize

		if end > len(imagesToDelete) {
			end = len(imagesToDelete)
		}

		imagesToDeleteChunks = append(imagesToDeleteChunks, imagesToDelete[i:end])
	}

	count := 0

	for _, imagesToDeleteChunk := range imagesToDeleteChunks {
		o3, err := ecrHandler.BatchDeleteImage(ctx, &ecrLib.BatchDeleteImageInput{
			RepositoryName: aws.String(repository),
			ImageIds:       imagesToDeleteChunk,
		})
		if err != nil {
			return err
		}

		if len(o3.Failures) > 0 {
			for _, failure := range o3.Failures {
				logger.Error("Unable to delete image %s because %s", failure.ImageId.ImageDigest, aws.ToString(failure.FailureReason))
			}
		}

		count += len(imagesToDeleteChunk)
	}

	logger.Info("deleted %d images for %s (%s)", count, repository, identifier)

	return nil
}
