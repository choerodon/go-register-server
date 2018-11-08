package apps

import "time"

//type GeneralInfo struct {
//	IpAddr                    string `xml:"ipAddr" json:"ipAddr"`
//	UsedMemory                string   `xml:"usedMemory" json:"usedMemory"`
//	NumOfCpu                  int   `xml:"numOfCpu" json:"numOfCpu"`
//	NamespaceOfRegisterServer string   `xml:"namespaceOfRegisterServer" json:"namespaceOfRegisterServer"`
//	NamespacesOfListeningOn   []string `xml:"namespacesOfListeningOn" json:"namespacesOfListeningOn"`
//}

type ApplicationResources struct {
	Applications *Applications `xml:"applications" json:"applications"`
}

type Applications struct {
	ApplicationList []*Application `xml:"application" json:"application"`
	AppsHashcode    string         `xml:"apps__hashcode" json:"apps__hashcode"`
	VersionsDelta   int            `xml:"versions__delta" json:"versions__delta"`
}

type Application struct {
	Name      string      `xml:"name" json:"name"`
	Instances []*Instance `xml:"instance" json:"instance"`
}

type EurekaPage struct {
	GeneralInfo        map[string]interface{}
	InstanceInfo       map[string]interface{}
	EurekaInstances    []*EurekaInstance
	AvailableRegisters []*Instance
	CurrentTime        time.Time
}

type EurekaInstance struct {
	Name                 string
	Available            []*Instance
	InAvailable          []*Instance
	AvailableInstances   []string
	InAvailableInstances []string
	AMIs                 string
	AvailabilityZones    int
}

// StatusType is an enum of the different statuses allowed by Eureka.
type StatusType string

// Supported statuses
const (
	UP           StatusType = "UP"
	DOWN         StatusType = "DOWN"
	STARTING     StatusType = "STARTING"
	OUTOFSERVICE StatusType = "OUT_OF_SERVICE"
	UNKNOWN      StatusType = "UNKNOWN"
)

type Instance struct {
	InstanceId       string            `xml:"instanceId" json:"instanceId"`
	HostName         string            `xml:"hostName" json:"hostName"`
	App              string            `xml:"app" json:"app"`
	IPAddr           string            `xml:"ipAddr" json:"ipAddr"`
	Status           StatusType        `xml:"status" json:"status"`
	OverriddenStatus StatusType        `xml:"overriddenstatus" json:"overriddenstatus"`
	Port             Port              `xml:"port" json:"port"`
	SecurePort       Port              `xml:"securePort" json:"securePort"`
	CountryId        uint64            `xml:"countryId" json:"countryId"`
	DataCenterInfo   DataCenterInfo    `xml:"dataCenterInfo" json:"dataCenterInfo"`
	LeaseInfo        LeaseInfo         `xml:"leaseInfo" json:"leaseInfo"`
	Metadata         map[string]string `xml:"metadata" json:"metadata"`
	HomePageUrl      string            `xml:"homePageUrl" json:"homePageUrl"`
	StatusPageUrl    string            `xml:"statusPageUrl" json:"statusPageUrl"`
	HealthCheckUrl   string            `xml:"healthCheckUrl" json:"healthCheckUrl"`
	VipAddress       string            `xml:"vipAddress" json:"vipAddress"`
	SecureVipAddress string            `xml:"secureVipAddress" json:"secureVipAddress"`

	IsCoordinatingDiscoveryServer bool `xml:"isCoordinatingDiscoveryServer" json:"isCoordinatingDiscoveryServer"`

	LastUpdatedTimestamp uint64 `xml:"lastUpdatedTimestamp" json:"lastUpdatedTimestamp"`
	LastDirtyTimestamp   uint64 `xml:"lastDirtyTimestamp" json:"lastDirtyTimestamp"`
	ActionType           string `xml:"actionType" json:"actionType"`
}

type Port struct {
	Enabled bool  `xml:"@enabled" json:"@enabled"`
	Port    int32 `xml:"$" json:"$"`
}

type DataCenterInfo struct {
	Name  string `xml:"name" json:"name"`
	Class string `xml:"@class" json:"@class"`
}

type LeaseInfo struct {
	RenewalIntervalInSecs uint   `xml:"renewalIntervalInSecs" json:"renewalIntervalInSecs"`
	DurationInSecs        uint   `xml:"durationInSecs" json:"durationInSecs"`
	RegistrationTimestamp uint64 `xml:"registrationTimestamp" json:"registrationTimestamp"`
	LastRenewalTimestamp  uint64 `xml:"lastRenewalTimestamp" json:"lastRenewalTimestamp"`
	EvictionTimestamp     uint64 `xml:"evictionTimestamp" json:"evictionTimestamp"`
	ServiceUpTimestamp    uint64 `xml:"serviceUpTimestamp" json:"serviceUpTimestamp"`
}

type InstanceMetadata struct {
	Class string `xml:"@class" json:"@class"`
}

type RefArray *[1]int
