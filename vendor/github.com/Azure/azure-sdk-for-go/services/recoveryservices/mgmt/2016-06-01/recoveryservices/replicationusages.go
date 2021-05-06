package recoveryservices

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
//
// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"context"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/tracing"
	"net/http"
)

// ReplicationUsagesClient is the recovery Services Client
type ReplicationUsagesClient struct {
	BaseClient
}

// NewReplicationUsagesClient creates an instance of the ReplicationUsagesClient client.
func NewReplicationUsagesClient(subscriptionID string) ReplicationUsagesClient {
	return NewReplicationUsagesClientWithBaseURI(DefaultBaseURI, subscriptionID)
}

// NewReplicationUsagesClientWithBaseURI creates an instance of the ReplicationUsagesClient client using a custom
// endpoint.  Use this when interacting with an Azure cloud that uses a non-standard base URI (sovereign clouds, Azure
// stack).
func NewReplicationUsagesClientWithBaseURI(baseURI string, subscriptionID string) ReplicationUsagesClient {
	return ReplicationUsagesClient{NewWithBaseURI(baseURI, subscriptionID)}
}

// List fetches the replication usages of the vault.
// Parameters:
// resourceGroupName - the name of the resource group where the recovery services vault is present.
// vaultName - the name of the recovery services vault.
func (client ReplicationUsagesClient) List(ctx context.Context, resourceGroupName string, vaultName string) (result ReplicationUsageList, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/ReplicationUsagesClient.List")
		defer func() {
			sc := -1
			if result.Response.Response != nil {
				sc = result.Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	req, err := client.ListPreparer(ctx, resourceGroupName, vaultName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "recoveryservices.ReplicationUsagesClient", "List", nil, "Failure preparing request")
		return
	}

	resp, err := client.ListSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "recoveryservices.ReplicationUsagesClient", "List", resp, "Failure sending request")
		return
	}

	result, err = client.ListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "recoveryservices.ReplicationUsagesClient", "List", resp, "Failure responding to request")
		return
	}

	return
}

// ListPreparer prepares the List request.
func (client ReplicationUsagesClient) ListPreparer(ctx context.Context, resourceGroupName string, vaultName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
		"vaultName":         autorest.Encode("path", vaultName),
	}

	const APIVersion = "2016-06-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/Subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.RecoveryServices/vaults/{vaultName}/replicationUsages", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// ListSender sends the List request. The method will close the
// http.Response Body if it receives an error.
func (client ReplicationUsagesClient) ListSender(req *http.Request) (*http.Response, error) {
	return client.Send(req, azure.DoRetryWithRegistration(client.Client))
}

// ListResponder handles the response to the List request. The method always
// closes the http.Response Body.
func (client ReplicationUsagesClient) ListResponder(resp *http.Response) (result ReplicationUsageList, err error) {
	err = autorest.Respond(
		resp,
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}
