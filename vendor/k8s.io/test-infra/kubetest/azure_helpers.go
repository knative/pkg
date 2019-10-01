/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	resources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

type AcsEngineAPIModel struct {
	Location   string            `json:"location,omitempty"`
	Name       string            `json:"name,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	APIVersion string            `json:"APIVersion"`

	Properties *Properties `json:"properties"`
}

type Properties struct {
	OrchestratorProfile     *OrchestratorProfile     `json:"orchestratorProfile,omitempty"`
	MasterProfile           *MasterProfile           `json:"masterProfile,omitempty"`
	AgentPoolProfiles       []*AgentPoolProfile      `json:"agentPoolProfiles,omitempty"`
	LinuxProfile            *LinuxProfile            `json:"linuxProfile,omitempty"`
	WindowsProfile          *WindowsProfile          `json:"windowsProfile,omitempty"`
	ServicePrincipalProfile *ServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`
	ExtensionProfiles       []map[string]string      `json:"extensionProfiles,omitempty"`
	CustomCloudProfile      *CustomCloudProfile      `json:"customCloudProfile,omitempty"`
	FeatureFlags            *FeatureFlags            `json:"featureFlags,omitempty"`
}

type ServicePrincipalProfile struct {
	ClientID string `json:"clientId,omitempty"`
	Secret   string `json:"secret,omitempty"`
}

type LinuxProfile struct {
	AdminUsername string `json:"adminUsername"`
	SSHKeys       *SSH   `json:"ssh"`
}

type SSH struct {
	PublicKeys []PublicKey `json:"publicKeys"`
}

type PublicKey struct {
	KeyData string `json:"keyData"`
}

type WindowsProfile struct {
	AdminUsername         string `json:"adminUsername,omitempty"`
	AdminPassword         string `json:"adminPassword,omitempty"`
	ImageVersion          string `json:"imageVersion,omitempty"`
	WindowsImageSourceURL string `json:"WindowsImageSourceUrl"`
	WindowsPublisher      string `json:"WindowsPublisher"`
	WindowsOffer          string `json:"WindowsOffer"`
	WindowsSku            string `json:"WindowsSku"`
	WindowsDockerVersion  string `json:"windowsDockerVersion"`
	SSHEnabled            bool   `json:"sshEnabled,omitempty"`
}

// KubernetesContainerSpec defines configuration for a container spec
type KubernetesContainerSpec struct {
	Name           string `json:"name,omitempty"`
	Image          string `json:"image,omitempty"`
	CPURequests    string `json:"cpuRequests,omitempty"`
	MemoryRequests string `json:"memoryRequests,omitempty"`
	CPULimits      string `json:"cpuLimits,omitempty"`
	MemoryLimits   string `json:"memoryLimits,omitempty"`
}

// KubernetesAddon defines a list of addons w/ configuration to include with the cluster deployment
type KubernetesAddon struct {
	Name       string                    `json:"name,omitempty"`
	Enabled    *bool                     `json:"enabled,omitempty"`
	Containers []KubernetesContainerSpec `json:"containers,omitempty"`
	Config     map[string]string         `json:"config,omitempty"`
	Data       string                    `json:"data,omitempty"`
}

type KubernetesConfig struct {
	CustomWindowsPackageURL      string            `json:"customWindowsPackageURL,omitempty"`
	CustomHyperkubeImage         string            `json:"customHyperkubeImage,omitempty"`
	CustomCcmImage               string            `json:"customCcmImage,omitempty"` // Image for cloud-controller-manager
	UseCloudControllerManager    *bool             `json:"useCloudControllerManager,omitempty"`
	NetworkPlugin                string            `json:"networkPlugin,omitempty"`
	PrivateAzureRegistryServer   string            `json:"privateAzureRegistryServer,omitempty"`
	AzureCNIURLLinux             string            `json:"azureCNIURLLinux,omitempty"`
	AzureCNIURLWindows           string            `json:"azureCNIURLWindows,omitempty"`
	Addons                       []KubernetesAddon `json:"addons,omitempty"`
	NetworkPolicy                string            `json:"networkPolicy,omitempty"`
	CloudProviderRateLimitQPS    float64           `json:"cloudProviderRateLimitQPS,omitempty"`
	CloudProviderRateLimitBucket int               `json:"cloudProviderRateLimitBucket,omitempty"`
	APIServerConfig              map[string]string `json:"apiServerConfig,omitempty"`
	KubernetesImageBase          string            `json:"kubernetesImageBase,omitempty"`
	ControllerManagerConfig      map[string]string `json:"controllerManagerConfig,omitempty"`
	KubeletConfig                map[string]string `json:"kubeletConfig,omitempty"`
}

type OrchestratorProfile struct {
	OrchestratorType    string            `json:"orchestratorType"`
	OrchestratorRelease string            `json:"orchestratorRelease"`
	KubernetesConfig    *KubernetesConfig `json:"kubernetesConfig,omitempty"`
}

type MasterProfile struct {
	Count          int                 `json:"count"`
	Distro         string              `json:"distro"`
	DNSPrefix      string              `json:"dnsPrefix"`
	VMSize         string              `json:"vmSize" validate:"required"`
	IPAddressCount int                 `json:"ipAddressCount,omitempty"`
	Extensions     []map[string]string `json:"extensions,omitempty"`
	OSDiskSizeGB   int                 `json:"osDiskSizeGB,omitempty" validate:"min=0,max=1023"`
}

type AgentPoolProfile struct {
	Name                   string              `json:"name"`
	Count                  int                 `json:"count"`
	Distro                 string              `json:"distro"`
	VMSize                 string              `json:"vmSize"`
	OSType                 string              `json:"osType,omitempty"`
	AvailabilityProfile    string              `json:"availabilityProfile"`
	IPAddressCount         int                 `json:"ipAddressCount,omitempty"`
	PreProvisionExtension  map[string]string   `json:"preProvisionExtension,omitempty"`
	Extensions             []map[string]string `json:"extensions,omitempty"`
	OSDiskSizeGB           int                 `json:"osDiskSizeGB,omitempty" validate:"min=0,max=1023"`
	EnableVMSSNodePublicIP bool                `json:"enableVMSSNodePublicIP,omitempty"`
}

type AzureClient struct {
	environment       azure.Environment
	subscriptionID    string
	deploymentsClient resources.DeploymentsClient
	groupsClient      resources.GroupsClient
}

type FeatureFlags struct {
	EnableIPv6DualStack bool `json:"enableIPv6DualStack,omitempty"`
}

// CustomCloudProfile defines configuration for custom cloud profile( for ex: Azure Stack)
type CustomCloudProfile struct {
	PortalURL string `json:"portalURL,omitempty"`
}

// AzureStackMetadataEndpoints defines configuration for Azure Stack
type AzureStackMetadataEndpoints struct {
	GalleryEndpoint string                            `json:"galleryEndpoint,omitempty"`
	GraphEndpoint   string                            `json:"graphEndpoint,omitempty"`
	PortalEndpoint  string                            `json:"portalEndpoint,omitempty"`
	Authentication  *AzureStackMetadataAuthentication `json:"authentication,omitempty"`
}

// AzureStackMetadataAuthentication defines configuration for Azure Stack
type AzureStackMetadataAuthentication struct {
	LoginEndpoint string   `json:"loginEndpoint,omitempty"`
	Audiences     []string `json:"audiences,omitempty"`
}

func (az *AzureClient) ValidateDeployment(ctx context.Context, resourceGroupName, deploymentName string, template, params *map[string]interface{}) (valid resources.DeploymentValidateResult, err error) {
	return az.deploymentsClient.Validate(ctx,
		resourceGroupName,
		deploymentName,
		resources.Deployment{
			Properties: &resources.DeploymentProperties{
				Template:   template,
				Parameters: params,
				Mode:       resources.Incremental,
			},
		})
}

func (az *AzureClient) DeployTemplate(ctx context.Context, resourceGroupName, deploymentName string, template, parameters *map[string]interface{}) (de resources.DeploymentExtended, err error) {
	future, err := az.deploymentsClient.CreateOrUpdate(
		ctx,
		resourceGroupName,
		deploymentName,
		resources.Deployment{
			Properties: &resources.DeploymentProperties{
				Template:   template,
				Parameters: parameters,
				Mode:       resources.Incremental,
			},
		})
	if err != nil {
		return de, fmt.Errorf("cannot create deployment: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, az.deploymentsClient.Client)
	if err != nil {
		return de, fmt.Errorf("cannot get the create deployment future response: %v", err)
	}

	return future.Result(az.deploymentsClient)
}

func (az *AzureClient) EnsureResourceGroup(ctx context.Context, name, location string, managedBy *string) (resourceGroup *resources.Group, err error) {
	var tags map[string]*string
	group, err := az.groupsClient.Get(ctx, name)
	if err == nil {
		tags = group.Tags
	}

	response, err := az.groupsClient.CreateOrUpdate(ctx, name, resources.Group{
		Name:      &name,
		Location:  &location,
		ManagedBy: managedBy,
		Tags:      tags,
	})
	if err != nil {
		return &response, err
	}

	return &response, nil
}

func (az *AzureClient) DeleteResourceGroup(ctx context.Context, groupName string) error {
	_, err := az.groupsClient.Get(ctx, groupName)
	if err == nil {
		future, err := az.groupsClient.Delete(ctx, groupName)
		if err != nil {
			return fmt.Errorf("cannot delete resource group %v: %v", groupName, err)
		}
		err = future.WaitForCompletionRef(ctx, az.groupsClient.Client)
		if err != nil {
			// Skip the teardown errors because of https://github.com/Azure/go-autorest/issues/357
			// TODO(feiskyer): fix the issue by upgrading go-autorest version >= v11.3.2.
			log.Printf("Warning: failed to delete resource group %q with error %v", groupName, err)
		}
	}
	return nil
}

func getOAuthConfig(env azure.Environment, subscriptionID, tenantID string) (*adal.OAuthConfig, error) {

	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	return oauthConfig, nil
}

func getAzureClient(env azure.Environment, subscriptionID, clientID, tenantID, clientSecret string) (*AzureClient, error) {
	oauthConfig, err := getOAuthConfig(env, subscriptionID, tenantID)
	if err != nil {
		return nil, err
	}

	armSpt, err := adal.NewServicePrincipalToken(*oauthConfig, clientID, clientSecret, env.ServiceManagementEndpoint)
	if err != nil {
		return nil, err
	}

	return getClient(env, subscriptionID, tenantID, armSpt), nil
}

func getClient(env azure.Environment, subscriptionID, tenantID string, armSpt *adal.ServicePrincipalToken) *AzureClient {
	c := &AzureClient{
		environment:    env,
		subscriptionID: subscriptionID,

		deploymentsClient: resources.NewDeploymentsClientWithBaseURI(env.ResourceManagerEndpoint, subscriptionID),
		groupsClient:      resources.NewGroupsClientWithBaseURI(env.ResourceManagerEndpoint, subscriptionID),
	}

	authorizer := autorest.NewBearerAuthorizer(armSpt)
	c.deploymentsClient.Authorizer = authorizer
	c.deploymentsClient.PollingDuration = 60 * time.Minute
	c.groupsClient.Authorizer = authorizer

	return c
}
