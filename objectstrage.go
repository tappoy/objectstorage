// This package provides a simple object storage interface
// for storing and retrieving objects.
//
// Wrapping around the OpenStack Object Storage API.
// See below for the original OpenStack package.
//   - `github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers`
//   - `github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects`
package objectstorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects"

	"github.com/tappoy/vault"
)

var (
	// The bucket already exists.
	ErrBucketAlreadyExists = errors.New("ErrBucketAlreadyExists")

	// The request could not be completed due to a conflict with the current state of the target resource.
	ErrConflict = errors.New("ErrConflict")

	// The specified bucket does not exist.
	ErrNoSuchBucket = errors.New("ErrNoSuchBucket")

	// The specified key does not exist.
	ErrNoSuchKey = errors.New("ErrNoSuchKey")

	// The specified key does not exist.
	ErrCannotCreateFile = errors.New("ErrCannotCreateFile")

	// Cannot open the file.
	ErrCannotOpenFile = errors.New("ErrCannotOpenFile")
)

type ObjectStorage struct {
	client *gophercloud.ServiceClient
	ctx    context.Context
}

// Return gophercloud error.
func apiError(err error) error {
	return fmt.Errorf("ErrGophercloud: %v", err)
}

// This is returned when listing objects in the container.
type ObjectInfo struct {
	// The size of the object in bytes.
	Bytes int64

	// The name of the object.
	Name string

	// The last modified time of the object.
	LastModified time.Time
}

// The options for creating a new ObjectStorage instance.
type Options struct {
	// The URL of the authentication service.
	AuthUrl string

	// The username of the user.
	Username string

	// The password of the user.
	Password string

	// The ID of the tenant.
	TenantId string

	// The name of the domain.
	// In ConoHa, it is the tenant name.
	DomainName string
}

var nilOptions = Options{}

func vaultError(err error) error {
	return fmt.Errorf("ErrVault: %v", err)
}

// AuthOptionsFromVault returns the Options from the vault.
//
// VaultKeys:
//   - OS_AUTH_URL
//   - OS_USERNAME
//   - OS_PASSWORD
//   - OS_TENANT_ID
//   - OS_DOMAIN_NAME
//
// Errors:
//   - "ErrVault: %v"
func AuthOptionsFromVault(password string, vaultDir string) (Options, error) {
	vaultClient, err := vault.NewVault(password, vaultDir)
	if err != nil {
		return nilOptions, vaultError(err)
	}

	authUrl, err := vaultClient.Get("OS_AUTH_URL")
	if err != nil {
		return nilOptions, vaultError(err)
	}

	username, err := vaultClient.Get("OS_USERNAME")
	if err != nil {
		return nilOptions, vaultError(err)
	}

	osPassword, err := vaultClient.Get("OS_PASSWORD")
	if err != nil {
		return nilOptions, vaultError(err)
	}

	tenantId, err := vaultClient.Get("OS_TENANT_ID")
	if err != nil {
		return nilOptions, vaultError(err)
	}

	domainName, err := vaultClient.Get("OS_DOMAIN_NAME")
	if err != nil {
		return nilOptions, vaultError(err)
	}

	return Options{
		AuthUrl:    authUrl,
		Username:   username,
		Password:   osPassword,
		TenantId:   tenantId,
		DomainName: domainName,
	}, nil
}

// Create a new ObjectStorage instance.
//
// Errors:
//   - "ErrGophercloud: %v"
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
		return nil, apiError(err)
	}

	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, apiError(err)
	}

	return &ObjectStorage{client: client, ctx: ctx}, nil
}

// List the containers in the account.
//
// Errors:
//   - "ErrGophercloud: %v"
func (ost *ObjectStorage) ListContainers(prefix string) ([]string, error) {
	listOpt := containers.ListOpts{
		Prefix: prefix,
	}

	pages, err := containers.List(ost.client, listOpt).AllPages(ost.ctx)
	if err != nil {
		return nil, apiError(err)
	}

	containerList, err := containers.ExtractInfo(pages)
	if err != nil {
		return nil, apiError(err)
	}

	var names []string
	for _, container := range containerList {
		names = append(names, container.Name)
	}

	return names, nil
}

// Create a new container.
//
// Errors:
//   - ErrBucketAlreadyExists
//   - "ErrGophercloud: %v"
func (ost *ObjectStorage) CreateContainer(name string) error {
	result := containers.Create(ost.ctx, ost.client, name, nil)
	if result.Err != nil {
		// check if BucketAlreadyExists
		if strings.Contains(result.Err.Error(), "BucketAlreadyExists") {
			return ErrBucketAlreadyExists
		} else {
			return apiError(result.Err)
		}
	}

	return nil
}

// Delete the container.
//
// Errors:
//   - ErrConflict
//   - "ErrGophercloud: %v"
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
			return apiError(result.Err)
		}
	}

	return nil
}

// Upload a file to the container.
//
// Errors:
//   - ErrCannotOpenFile
//   - ErrNoSuchBucket
//   - "ErrGophercloud: %v"
func (ost *ObjectStorage) UploadFile(containerName string, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return ErrCannotOpenFile
	}
	defer file.Close()

	return ost.Upload(containerName, filepath.Base(path), file)
}

// Upload to the container from io.Reader.
//
// Errors:
//   - ErrNoSuchBucket
//   - "ErrGophercloud: %v"
func (ost *ObjectStorage) Upload(containerName string, objectName string, reader io.Reader) error {
	opt := objects.CreateOpts{
		Content: reader,
	}

	result := objects.Create(ost.ctx, ost.client, containerName, objectName, opt)

	if result.Err != nil {
		if strings.Contains(result.Err.Error(), "NoSuchBucket") {
			return ErrNoSuchBucket
		} else {
			return apiError(result.Err)
		}
	}

	return nil
}

// Download the object from the container to the path.
//
// Errors:
//   - ErrCannotCreateFile
//   - ErrNoSuchBucket
//   - ErrNoSuchKey
//   - "ErrGophercloud: %v"
func (ost *ObjectStorage) DownloadFile(containerName string, objectName string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return ErrCannotCreateFile
	}
	defer file.Close()

	result := objects.Download(ost.ctx, ost.client, containerName, objectName, nil)
	if result.Err != nil {
		if strings.Contains(result.Err.Error(), "NoSuchBucket") {
			return ErrNoSuchBucket
		} else if strings.Contains(result.Err.Error(), "NoSuchKey") {
			return ErrNoSuchKey
		} else {
			return apiError(result.Err)
		}
	}

	content, err := result.ExtractContent()
	if err != nil {
		return apiError(err)
	}
	_, err = file.Write(content)
	if err != nil {
		return apiError(err)
	}

	return nil
}

// Delete the object from the container.
//
// Errors:
//   - ErrNoSuchBucket
//   - "ErrGophercloud: %v"
func (ost *ObjectStorage) DeleteObject(containerName, objectName string) error {
	result := objects.Delete(ost.ctx, ost.client, containerName, objectName, nil)
	if result.Err != nil {
		if strings.Contains(result.Err.Error(), "NoSuchBucket") {
			return ErrNoSuchBucket
		} else if strings.Contains(result.Err.Error(), "NoSuchKey") {
			return ErrNoSuchKey
		} else {
			return apiError(result.Err)
		}
	}

	return nil
}

// List the objects in the container.
//
// Errors:
//   - ErrNoSuchBucket
//   - "ErrGophercloud: %v"
func (ost *ObjectStorage) ListObjects(containerName string) ([]ObjectInfo, error) {
	listOpt := objects.ListOpts{}

	pages, err := objects.List(ost.client, containerName, listOpt).AllPages(ost.ctx)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchBucket") {
			return nil, ErrNoSuchBucket
		} else {
			return nil, apiError(err)
		}
	}

	objectList, err := objects.ExtractInfo(pages)
	if err != nil {
		return nil, apiError(err)
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
