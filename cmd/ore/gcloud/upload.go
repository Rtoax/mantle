// Copyright 2015 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcloud

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/storage/v1"

	"github.com/flatcar-linux/mantle/platform/api/gcloud"
	"github.com/flatcar-linux/mantle/sdk"
)

var (
	cmdUpload = &cobra.Command{
		Use:   "upload",
		Short: "Upload os image",
		Long:  "Upload os image to Google Storage bucket and create image in GCE. Intended for use in SDK.",
		Run:   runUpload,
	}

	uploadBucket    string
	uploadImageName string
	uploadBoard     string
	uploadFile      string
	uploadForce     bool
	uploadPublic    bool
)

func init() {
	build := sdk.BuildRoot()
	cmdUpload.Flags().StringVar(&uploadBucket, "bucket", "gs://users.developer.core-os.net", "gs://bucket/prefix/ prefix defaults to $USER")
	cmdUpload.Flags().StringVar(&uploadImageName, "name", "", "name for uploaded image, defaults to COREOS_VERSION")
	cmdUpload.Flags().StringVar(&uploadBoard, "board", "amd64-usr", "board used for naming with default prefix only")
	cmdUpload.Flags().StringVar(&uploadFile, "file",
		build+"/images/amd64-usr/latest/flatcar_production_gce.tar.gz",
		"path_to_flatcar_image (build with: ./image_to_vm.sh --format=gce ...)")
	cmdUpload.Flags().BoolVar(&uploadForce, "force", false, "overwrite existing GS and GCE images without prompt")
	cmdUpload.Flags().BoolVar(&uploadPublic, "public", false, "Set public ACLs on image")
	GCloud.AddCommand(cmdUpload)
}

func runUpload(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "Unrecognized args in plume upload cmd: %v\n", args)
		os.Exit(2)
	}

	// if an image name is unspecified try to use version.txt
	if uploadImageName == "" {
		ver, err := sdk.VersionsFromDir(filepath.Dir(uploadFile))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to get version from image directory, provide a -name flag or include a version.txt in the image directory: %v\n", err)
			os.Exit(1)
		}
		uploadImageName = ver.Version
	}

	gsURL, err := url.Parse(uploadBucket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if gsURL.Scheme != "gs" {
		fmt.Fprintf(os.Stderr, "URL missing gs:// scheme prefix: %v\n", uploadBucket)
		os.Exit(1)
	}
	if gsURL.Host == "" {
		fmt.Fprintf(os.Stderr, "URL missing bucket name %v\n", uploadBucket)
		os.Exit(1)
	}
	// if prefix not specified default name to gs://bucket/$USER/$BOARD/$VERSION
	if gsURL.Path == "" {
		if user := os.Getenv("USER"); user != "" {
			gsURL.Path = "/" + os.Getenv("USER")
			gsURL.Path += "/" + uploadBoard
		}
	}

	uploadBucket = gsURL.Host
	uploadImageName = strings.TrimPrefix(gsURL.Path+"/"+uploadImageName, "/")
	// create equivalent image names for GS and GCE
	imageNameGCE := gceSanitize(uploadImageName)
	imageNameGS := uploadImageName + ".tar.gz"

	storageAPI, err := storage.New(api.Client())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Storage client failed: %v\n", err)
		os.Exit(1)
	}

	// check if this file is already uploaded and give option to skip
	alreadyExists, err := fileQuery(storageAPI, uploadBucket, imageNameGS)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Uploading image failed: %v\n", err)
		os.Exit(1)
	}

	if alreadyExists && !uploadForce {
		var ans string
		fmt.Printf("File %v already exists on Google Storage. Overwrite? (y/n):", imageNameGS)
		if _, err = fmt.Scan(&ans); err != nil {
			fmt.Fprintf(os.Stderr, "Scanning overwrite input: %v", err)
			os.Exit(1)
		}
		switch ans {
		case "y", "Y", "yes":
			fmt.Println("Overriding existing file...")
			err = writeFile(storageAPI, uploadBucket, uploadFile, imageNameGS)
		default:
			fmt.Println("Skipped file upload")
		}
	} else {
		err = writeFile(storageAPI, uploadBucket, uploadFile, imageNameGS)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Uploading image failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Creating image in GCE: %v...\n", imageNameGCE)

	// create image on gce
	storageSrc := fmt.Sprintf("https://storage.googleapis.com/%v/%v", uploadBucket, imageNameGS)
	_, pending, err := api.CreateImage(&gcloud.ImageSpec{
		Name:        imageNameGCE,
		SourceImage: storageSrc,
	}, uploadForce)
	if err == nil {
		err = pending.Wait()
	}

	// if image already exists ask to delete and try again
	if err != nil && strings.HasSuffix(err.Error(), "alreadyExists") {
		var ans string
		fmt.Printf("Image %v already exists on GCE. Overwrite? (y/n):", imageNameGCE)
		if _, err = fmt.Scan(&ans); err != nil {
			fmt.Fprintf(os.Stderr, "Scanning overwrite input: %v", err)
			os.Exit(1)
		}
		switch ans {
		case "y", "Y", "yes":
			fmt.Println("Overriding existing image...")
			_, pending, err = api.CreateImage(&gcloud.ImageSpec{
				Name:        imageNameGCE,
				SourceImage: storageSrc,
			}, true)
			if err == nil {
				err = pending.Wait()
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Creating GCE image failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Image %v sucessfully created in GCE\n", imageNameGCE)
		default:
			fmt.Println("Skipped GCE image creation")
		}

		// If requested, set the image ACL to public
		if uploadPublic {
			fmt.Printf("Setting image to have public access: %v\n", imageNameGCE)
			err = api.SetImagePublic(imageNameGCE)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Marking GCE image with public ACLs failed: %v\n", err)
				os.Exit(1)
			}
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Creating GCE image failed: %v\n", err)
		os.Exit(1)
	}
}

// Converts an image name from Google Storage to an equivalent GCE image
// name. NOTE: Not a fully generlized sanitizer for GCE. Designed for
// the default version.txt name (ex: 633.1.0+2015-03-31-1538). See:
// https://godoc.org/google.golang.org/api/compute/v1#Image
func gceSanitize(name string) string {
	if name == "" {
		return name
	}

	// remove incompatible chars from version.txt
	name = strings.Replace(name, ".", "-", -1)
	name = strings.Replace(name, "+", "-", -1)

	// remove forward slashes likely from prefix
	name = strings.Replace(name, "/", "-", -1)

	// ensure name starts with [a-z]
	char := name[0]
	if char >= 'a' && char <= 'z' {
		return name
	}
	if char >= 'A' && char <= 'Z' {
		return strings.ToLower(name[:1]) + name[1:]
	}
	return "v" + name
}

// Write file to Google Storage
func writeFile(api *storage.Service, bucket, filename, destname string) error {
	fmt.Printf("Writing %v to gs://%v ...\n", filename, bucket)
	fmt.Printf("(Sometimes this takes a few minutes)\n")

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	req := api.Objects.Insert(bucket, &storage.Object{
		Name:        destname,
		ContentType: "application/x-gzip",
	})
	req.PredefinedAcl("authenticatedRead")
	req.Media(file)

	if _, err := req.Do(); err != nil {
		return err
	}

	fmt.Printf("Upload successful!\n")
	return nil
}

// Test if file exists in Google Storage
func fileQuery(api *storage.Service, bucket, name string) (bool, error) {
	req := api.Objects.Get(bucket, name)
	if _, err := req.Do(); err != nil {
		if e, ok := err.(*googleapi.Error); ok && e.Code == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
