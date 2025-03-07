// Copyright 2016 CoreOS, Inc.
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

package azure

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/classic/management"
	"github.com/Azure/azure-sdk-for-go/services/classic/management/storageservice"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-01-01/storage"
)

var (
	azureImageURL = "services/images"
)

func (a *API) GetStorageServiceKeys(account string) (storageservice.GetStorageServiceKeysResponse, error) {
	return storageservice.NewClient(a.client).GetStorageServiceKeys(account)
}

func (a *API) GetStorageServiceKeysARM(account, resourceGroup string) (storage.AccountListKeysResult, error) {
	return a.accClient.ListKeys(context.TODO(), resourceGroup, account, storage.Kerb)
}

// https://msdn.microsoft.com/en-us/library/azure/jj157192.aspx
func (a *API) AddOSImage(md *OSImage) error {
	data, err := xml.Marshal(md)
	if err != nil {
		return err
	}

	op, err := a.client.SendAzurePostRequest(azureImageURL, data)
	if err != nil {
		return err
	}

	return a.client.WaitForOperation(op, nil)
}

func (a *API) OSImageExists(name string) (bool, error) {
	url := fmt.Sprintf("%s/%s", azureImageURL, name)
	response, err := a.client.SendAzureGetRequest(url)
	if err != nil {
		if management.IsResourceNotFoundError(err) {
			return false, nil
		}

		return false, err
	}

	var image OSImage

	if err := xml.Unmarshal(response, &image); err != nil {
		return false, err
	}

	if image.Name == name {
		return true, nil
	}

	return false, nil
}

func (a *API) UrlOfBlob(account, container, blob string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   fmt.Sprintf("%s.blob.%s", account, a.opts.StorageEndpointSuffix),
		Path:   path.Join(container, blob),
	}
}

func (a *API) CreateStorageAccount(resourceGroup string) (string, error) {
	// Only lower-case letters & numbers allowed in storage account names
	name := strings.Replace(randomName("kolasa"), "-", "", -1)
	parameters := storage.AccountCreateParameters{
		Sku: &storage.Sku{
			Name: "Standard_LRS",
		},
		Kind:     "StorageV2",
		Location: &a.opts.Location,
	}
	plog.Infof("Creating StorageAccount %s", name)
	future, err := a.accClient.Create(context.TODO(), resourceGroup, name, parameters)
	if err != nil {
		return "", fmt.Errorf("start creating storage account: %v", err)
	}
	err = future.WaitForCompletionRef(context.TODO(), a.accClient.Client)
	if err != nil {
		return "", fmt.Errorf("finish creating storage account: %v", err)
	}
	_, err = future.Result(a.accClient)
	if err != nil {
		return "", fmt.Errorf("creating storage account: %v", err)
	}
	return name, nil
}
