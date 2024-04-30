# Package
`github.com/tappoy/objectstorage`

# About
This golang package provides a simple object storage interface for storing and retrieving objects.

# Features
- Wrapping around the OpenStack Object Storage API. See below for the original OpenStack package.
  - `github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers`
  - `github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects`

# Structs
```go
type Options struct {
	AuthUrl    string
	Username   string
	Password   string
	TenantId   string
	DomainName string
}

type ObjectInfo struct {
	Bytes        int64
	Name         string
	LastModified time.Time
}
```

# Functions
- `NewObjectStorage(options Options) (*ObjectStorage, error)` - Create a new object storage client.
- `(ost *ObjectStorage) ListContainers(prefix string) ([]string, error)` - List containers.
- `(ost *ObjectStorage) CreateContainer(name string) error` - Create a container.
- `(ost *ObjectStorage) DeleteContainer(name string) error` - Delete a container.
- `(ost *ObjectStorage) UploadFile(containerName string, path string) error` - Upload a file to a container.
- `(ost *ObjectStorage) DownloadFile(containerName string, objectName string, path string) error` - Download a file from a container.
- `(ost *ObjectStorage) DeleteObject(containerName, objectName string) error` - Delete an object.
- `(ost *ObjectStorage) ListObjects(containerName string) ([]ObjectInfo, error)` - List objects in a container.

# Errors
- `ErrBucketAlreadyExists` - The bucket already exists.
- `ErrConflict` - The request could not be completed due to a conflict with the current state of the target resource.
- `ErrNoSuchBucket` - The specified bucket does not exist.
- `ErrNoSuchKey` - The specified key does not exist.

# License
[LGPL-3.0](LICENSE)

# Author
[tappoy](https://github.com/tappoy)
