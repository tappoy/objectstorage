package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects"
)

// Errors
var (
	ErrBucketAlreadyExists = errors.New("ErrBucketAlreadyExists")
	ErrConflict            = errors.New("ErrConflict")
	ErrNoSuchBucket        = errors.New("ErrNoSuchBucket")
	ErrNoSuchKey           = errors.New("ErrNoSuchKey")
)

type ObjectStorage struct {
	client *gophercloud.ServiceClient
	ctx    context.Context
}

type ObjectInfo struct {
	Bytes        int64
	Name         string
	LastModified time.Time
}

type Options struct {
	AuthUrl    string
	Username   string
	Password   string
	TenantId   string
	DomainName string
}

func NewObjectStorage(option Options) (*ObjectStorage, error) {
	ctx := context.Background()

	opts := gophercloud.AuthOptions{
		IdentityEndpoint: option.AuthUrl,
		Username:         option.Username,
		Password:         option.Password,
		TenantID:         option.TenantId,
		DomainName:       option.DomainName,
	}

	provider, err := openstack.AuthenticatedClient(ctx, opts)
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	return &ObjectStorage{client: client, ctx: ctx}, nil
}

func (ost *ObjectStorage) ListContainers(prefix string) ([]string, error) {
	listOpt := containers.ListOpts{
		Prefix: prefix,
	}

	pages, err := containers.List(ost.client, listOpt).AllPages(ost.ctx)
	if err != nil {
		return nil, err
	}

	containerList, err := containers.ExtractInfo(pages)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, container := range containerList {
		names = append(names, container.Name)
	}

	return names, nil
}

func (ost *ObjectStorage) CreateContainer(name string) error {
	result := containers.Create(ost.ctx, ost.client, name, nil)
	if result.Err != nil {
		// check if BucketAlreadyExists
		if strings.Contains(result.Err.Error(), "BucketAlreadyExists") {
			return ErrBucketAlreadyExists
		} else {
			return result.Err
		}
	}

	return nil
}

func (ost *ObjectStorage) DeleteContainer(name string) error {
	// check if container has objects
	c, err := ost.ListContainers(name)
	if err != nil {
		return err
	}
	if len(c) > 0 {
		return ErrConflict
	}
	result := containers.Delete(ost.ctx, ost.client, name)
	if result.Err != nil {
		if strings.Contains(result.Err.Error(), "NoSuchBucket") {
			return ErrNoSuchBucket
		} else if strings.Contains(result.Err.Error(), "but got 409 instead") {
			return ErrConflict
		} else {
			return result.Err
		}
	}

	return nil
}

func (ost *ObjectStorage) UploadFile(containerName string, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	opt := objects.CreateOpts{
		Content: file,
	}

	fileName := filepath.Base(path)

	result := objects.Create(ost.ctx, ost.client, containerName, fileName, opt)

	if result.Err != nil {
		if strings.Contains(result.Err.Error(), "NoSuchBucket") {
			return ErrNoSuchBucket
		} else {
			return result.Err
		}
	}

	return nil
}

func (ost *ObjectStorage) DownloadFile(containerName string, objectName string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	result := objects.Download(ost.ctx, ost.client, containerName, objectName, nil)
	if result.Err != nil {
		if strings.Contains(result.Err.Error(), "NoSuchBucket") {
			return ErrNoSuchBucket
		} else if strings.Contains(result.Err.Error(), "NoSuchKey") {
			return ErrNoSuchKey
		} else {
			return result.Err
		}
	}

	content, err := result.ExtractContent()
	if err != nil {
		return err
	}
	_, err = file.Write(content)
	if err != nil {
		return err
	}

	return nil
}

func (ost *ObjectStorage) DeleteObject(containerName, objectName string) error {
	result := objects.Delete(ost.ctx, ost.client, containerName, objectName, nil)
	if result.Err != nil {
		if strings.Contains(result.Err.Error(), "NoSuchBucket") {
			return ErrNoSuchBucket
		} else if strings.Contains(result.Err.Error(), "NoSuchKey") {
			return ErrNoSuchKey
		} else {
			return result.Err
		}
	}

	return nil
}

func (ost *ObjectStorage) ListObjects(containerName string) ([]ObjectInfo, error) {
	listOpt := objects.ListOpts{}

	pages, err := objects.List(ost.client, containerName, listOpt).AllPages(ost.ctx)
	if err != nil {
		return nil, err
	}

	objectList, err := objects.ExtractInfo(pages)
	if err != nil {
		return nil, err
	}

	var result []ObjectInfo
	for _, object := range objectList {
		result = append(result, ObjectInfo{
			Bytes:        object.Bytes,
			Name:         object.Name,
			LastModified: object.LastModified,
		})
	}

	return result, nil
}
