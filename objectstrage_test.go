package objectstorage

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

// test main
func TestMain(m *testing.M) {
	m.Run()
}

func TestAll(t *testing.T) {
	option := Options{
		AuthUrl:    os.Getenv("OS_AUTH_URL"),
		Username:   os.Getenv("OS_USERNAME"),
		Password:   os.Getenv("OS_PASSWORD"),
		TenantId:   os.Getenv("OS_TENANT_ID"),
		DomainName: os.Getenv("OS_DOMAIN_NAME"),
	}

	fmt.Printf("option: %v\n", option)

	ost, err := NewObjectStorage(option)

	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Token: ", ost.client.Token())

	// List containers
	fmt.Println("List containers:")
	list, err := ost.ListContainers("conoha_")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(list)

	// add container
	fmt.Println("Add container:")
	{
		fmt.Println("CreateContainer")
		err := ost.CreateContainer("conoha_container")
		if err != nil && err != ErrBucketAlreadyExists {
			fmt.Printf("DeleteContainer: %v\n", err)
			t.Error(err)
		}
		fmt.Println("CreateContainer2")
		err = ost.CreateContainer("conoha_container2")
		if err != nil && err != ErrBucketAlreadyExists {
			fmt.Printf("CreateContainer2: %v\n", err)
			t.Error(err)
		}

		fmt.Println("DeleteContainer")
		err = ost.DeleteContainer("conoha_container")
		if err != nil && err != ErrConflict {
			fmt.Printf("DeleteContainer: %v\n", err)
			t.Error(err)
		}

		fmt.Println("DeleteContainer")
		err = ost.DeleteContainer("not_exist_container")
		if err != nil && err != ErrNoSuchBucket {
			fmt.Printf("DeleteContainer: %v\n", err)
			t.Error(err)
		}
	}

	// clean up test file
	ulfile := "./tmp/ulfile"
	dlfile := "./tmp/dlfile"
	os.Remove(ulfile)
	os.Remove(dlfile)

	// make upload file
	testText := []byte("object strage test\n")
	err = ioutil.WriteFile(ulfile, testText, 0644)
	if err != nil {
		fmt.Printf("Can't make file: %v\n", err)
		t.Error(err)
	}

	// upload file
	fmt.Println("Add object:")
	err = ost.UploadFile("conoha_container2", ulfile)
	if err != nil {
		fmt.Printf("UploadFile: %v\n", err)
		t.Error(err)
	}

	// download file
	fmt.Println("Download object:")
	err = ost.DownloadFile("conoha_container2", "ulfile", dlfile)
	if err != nil {
		fmt.Printf("DownloadFile: %v\n", err)
		t.Error(err)
	}

	// publish container
	// {
	// 	fmt.Println("Publish container:")
	// 	optstr := new(string)
	// 	*optstr = ".r:*"
	// 	opt := containers.UpdateOpts{
	// 		ContainerRead: optstr,
	// 	}
	// 	result := containers.Update(ost.ctx, ost.client, "conoha_container", opt)
	// 	if result.Err != nil {
	// 		fmt.Printf("%+vn", result.Err.Error())
	// 		os.Exit(1)
	// 	} else {
	// 		fmt.Printf("%+vn", result)
	// 	}
	// }

	// list objects
	fmt.Println("List objects:")
	objlist, err := ost.ListObjects("conoha_container")
	if err != nil {
		fmt.Printf("ListObjects: %v\n", err)
		t.Error(err)
	}
	for _, obj := range objlist {
		fmt.Println("----------------")
		fmt.Println("Name: ", obj.Name)
		fmt.Println("Bytes: ", obj.Bytes)
		fmt.Println("LastModified: ", obj.LastModified)
	}

	// delete object
	fmt.Println("Delete object:")
	err = ost.DeleteObject("conoha_container2", "ulfile")
	if err != nil {
		fmt.Printf("DeleteFile: %v\n", err)
		t.Error(err)
	}

}
