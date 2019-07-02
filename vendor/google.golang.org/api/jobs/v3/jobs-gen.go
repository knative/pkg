// Package jobs provides access to the Cloud Talent Solution API.
//
// See https://cloud.google.com/talent-solution/job-search/docs/
//
// Usage example:
//
//   import "google.golang.org/api/jobs/v3"
//   ...
//   jobsService, err := jobs.New(oauthHttpClient)
package jobs // import "google.golang.org/api/jobs/v3"

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	context "golang.org/x/net/context"
	ctxhttp "golang.org/x/net/context/ctxhttp"
	gensupport "google.golang.org/api/gensupport"
	googleapi "google.golang.org/api/googleapi"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Always reference these packages, just in case the auto-generated code
// below doesn't.
var _ = bytes.NewBuffer
var _ = strconv.Itoa
var _ = fmt.Sprintf
var _ = json.NewDecoder
var _ = io.Copy
var _ = url.Parse
var _ = gensupport.MarshalJSON
var _ = googleapi.Version
var _ = errors.New
var _ = strings.Replace
var _ = context.Canceled
var _ = ctxhttp.Do

const apiId = "jobs:v3"
const apiName = "jobs"
const apiVersion = "v3"
const basePath = "https://jobs.googleapis.com/"

// OAuth2 scopes used by this API.
const (
	// View and manage your data across Google Cloud Platform services
	CloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

	// Manage job postings
	JobsScope = "https://www.googleapis.com/auth/jobs"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Projects = NewProjectsService(s)
	return s, nil
}

type Service struct {
	client    *http.Client
	BasePath  string // API endpoint base URL
	UserAgent string // optional additional User-Agent fragment

	Projects *ProjectsService
}

func (s *Service) userAgent() string {
	if s.UserAgent == "" {
		return googleapi.UserAgent
	}
	return googleapi.UserAgent + " " + s.UserAgent
}

func NewProjectsService(s *Service) *ProjectsService {
	rs := &ProjectsService{s: s}
	rs.Companies = NewProjectsCompaniesService(s)
	rs.Jobs = NewProjectsJobsService(s)
	return rs
}

type ProjectsService struct {
	s *Service

	Companies *ProjectsCompaniesService

	Jobs *ProjectsJobsService
}

func NewProjectsCompaniesService(s *Service) *ProjectsCompaniesService {
	rs := &ProjectsCompaniesService{s: s}
	return rs
}

type ProjectsCompaniesService struct {
	s *Service
}

func NewProjectsJobsService(s *Service) *ProjectsJobsService {
	rs := &ProjectsJobsService{s: s}
	return rs
}

type ProjectsJobsService struct {
	s *Service
}

// ApplicationInfo: Application related details of a job posting.
type ApplicationInfo struct {
	// Emails: Optional but at least one of uris,
	// emails or instruction must be
	// specified.
	//
	// Use this field to specify email address(es) to which resumes
	// or
	// applications can be sent.
	//
	// The maximum number of allowed characters for each entry is 255.
	Emails []string `json:"emails,omitempty"`

	// Instruction: Optional but at least one of uris,
	// emails or instruction must be
	// specified.
	//
	// Use this field to provide instructions, such as "Mail your
	// application
	// to ...", that a candidate can follow to apply for the job.
	//
	// This field accepts and sanitizes HTML input, and also accepts
	// bold, italic, ordered list, and unordered list markup tags.
	//
	// The maximum number of allowed characters is 3,000.
	Instruction string `json:"instruction,omitempty"`

	// Uris: Optional but at least one of uris,
	// emails or instruction must be
	// specified.
	//
	// Use this URI field to direct an applicant to a website, for example
	// to
	// link to an online application form.
	//
	// The maximum number of allowed characters for each entry is 2,000.
	Uris []string `json:"uris,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Emails") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Emails") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *ApplicationInfo) MarshalJSON() ([]byte, error) {
	type NoMethod ApplicationInfo
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// BatchDeleteJobsRequest: Input only.
//
// Batch delete jobs request.
type BatchDeleteJobsRequest struct {
	// Filter: Required.
	//
	// The filter string specifies the jobs to be deleted.
	//
	// Supported operator: =, AND
	//
	// The fields eligible for filtering are:
	//
	// * `companyName` (Required)
	// * `requisitionId` (Required)
	//
	// Sample Query: companyName = "projects/api-test-project/companies/123"
	// AND
	// requisitionId = "req-1"
	Filter string `json:"filter,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Filter") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Filter") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *BatchDeleteJobsRequest) MarshalJSON() ([]byte, error) {
	type NoMethod BatchDeleteJobsRequest
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// BucketRange: Represents starting and ending value of a range in
// double.
type BucketRange struct {
	// From: Starting value of the bucket range.
	From float64 `json:"from,omitempty"`

	// To: Ending value of the bucket range.
	To float64 `json:"to,omitempty"`

	// ForceSendFields is a list of field names (e.g. "From") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "From") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *BucketRange) MarshalJSON() ([]byte, error) {
	type NoMethod BucketRange
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

func (s *BucketRange) UnmarshalJSON(data []byte) error {
	type NoMethod BucketRange
	var s1 struct {
		From gensupport.JSONFloat64 `json:"from"`
		To   gensupport.JSONFloat64 `json:"to"`
		*NoMethod
	}
	s1.NoMethod = (*NoMethod)(s)
	if err := json.Unmarshal(data, &s1); err != nil {
		return err
	}
	s.From = float64(s1.From)
	s.To = float64(s1.To)
	return nil
}

// BucketizedCount: Represents count of jobs within one bucket.
type BucketizedCount struct {
	// Count: Number of jobs whose numeric field value fall into `range`.
	Count int64 `json:"count,omitempty"`

	// Range: Bucket range on which histogram was performed for the numeric
	// field,
	// that is, the count represents number of jobs in this range.
	Range *BucketRange `json:"range,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Count") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Count") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *BucketizedCount) MarshalJSON() ([]byte, error) {
	type NoMethod BucketizedCount
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CommuteFilter: Input only.
//
// Parameters needed for commute search.
type CommuteFilter struct {
	// AllowImpreciseAddresses: Optional.
	// If `true`, jobs without street level addresses may also be
	// returned.
	// For city level addresses, the city center is used. For state and
	// coarser
	// level addresses, text matching is used.
	// If this field is set to `false` or is not specified, only jobs that
	// include
	// street level addresses will be returned by commute search.
	AllowImpreciseAddresses bool `json:"allowImpreciseAddresses,omitempty"`

	// CommuteMethod: Required.
	//
	// The method of transportation for which to calculate the commute time.
	//
	// Possible values:
	//   "COMMUTE_METHOD_UNSPECIFIED" - Commute method is not specified.
	//   "DRIVING" - Commute time is calculated based on driving time.
	//   "TRANSIT" - Commute time is calculated based on public transit
	// including bus, metro,
	// subway, etc.
	CommuteMethod string `json:"commuteMethod,omitempty"`

	// DepartureTime: Optional.
	//
	// The departure time used to calculate traffic impact, represented
	// as
	// .google.type.TimeOfDay in local time zone.
	//
	// Currently traffic model is restricted to hour level resolution.
	DepartureTime *TimeOfDay `json:"departureTime,omitempty"`

	// RoadTraffic: Optional.
	//
	// Specifies the traffic density to use when caculating commute time.
	//
	// Possible values:
	//   "ROAD_TRAFFIC_UNSPECIFIED" - Road traffic situation is not
	// specified.
	//   "TRAFFIC_FREE" - Optimal commute time without considering any
	// traffic impact.
	//   "BUSY_HOUR" - Commute time calculation takes in account the peak
	// traffic impact.
	RoadTraffic string `json:"roadTraffic,omitempty"`

	// StartCoordinates: Required.
	//
	// The latitude and longitude of the location from which to calculate
	// the
	// commute time.
	StartCoordinates *LatLng `json:"startCoordinates,omitempty"`

	// TravelDuration: Required.
	//
	// The maximum travel time in seconds. The maximum allowed value is
	// `3600s`
	// (one hour). Format is `123s`.
	TravelDuration string `json:"travelDuration,omitempty"`

	// ForceSendFields is a list of field names (e.g.
	// "AllowImpreciseAddresses") to unconditionally include in API
	// requests. By default, fields with empty values are omitted from API
	// requests. However, any non-pointer, non-interface field appearing in
	// ForceSendFields will be sent to the server regardless of whether the
	// field is empty or not. This may be used to include empty fields in
	// Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "AllowImpreciseAddresses")
	// to include in API requests with the JSON null value. By default,
	// fields with empty values are omitted from API requests. However, any
	// field with an empty value appearing in NullFields will be sent to the
	// server as null. It is an error if a field in this list has a
	// non-empty value. This may be used to include null fields in Patch
	// requests.
	NullFields []string `json:"-"`
}

func (s *CommuteFilter) MarshalJSON() ([]byte, error) {
	type NoMethod CommuteFilter
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CommuteInfo: Output only.
//
// Commute details related to this job.
type CommuteInfo struct {
	// JobLocation: Location used as the destination in the commute
	// calculation.
	JobLocation *Location `json:"jobLocation,omitempty"`

	// TravelDuration: The number of seconds required to travel to the job
	// location from the
	// query location. A duration of 0 seconds indicates that the job is
	// not
	// reachable within the requested duration, but was returned as part of
	// an
	// expanded query.
	TravelDuration string `json:"travelDuration,omitempty"`

	// ForceSendFields is a list of field names (e.g. "JobLocation") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "JobLocation") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *CommuteInfo) MarshalJSON() ([]byte, error) {
	type NoMethod CommuteInfo
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// Company: A Company resource represents a company in the service. A
// company is the
// entity that owns job postings, that is, the hiring entity responsible
// for
// employing applicants for the job position.
type Company struct {
	// CareerSiteUri: Optional.
	//
	// The URI to employer's career site or careers page on the employer's
	// web
	// site, for example, "https://careers.google.com".
	CareerSiteUri string `json:"careerSiteUri,omitempty"`

	// DerivedInfo: Output only. Derived details about the company.
	DerivedInfo *CompanyDerivedInfo `json:"derivedInfo,omitempty"`

	// DisplayName: Required.
	//
	// The display name of the company, for example, "Google, LLC".
	DisplayName string `json:"displayName,omitempty"`

	// EeoText: Optional.
	//
	// Equal Employment Opportunity legal disclaimer text to be
	// associated with all jobs, and typically to be displayed in
	// all
	// roles.
	//
	// The maximum number of allowed characters is 500.
	EeoText string `json:"eeoText,omitempty"`

	// ExternalId: Required.
	//
	// Client side company identifier, used to uniquely identify
	// the
	// company.
	//
	// The maximum number of allowed characters is 255.
	ExternalId string `json:"externalId,omitempty"`

	// HeadquartersAddress: Optional.
	//
	// The street address of the company's main headquarters, which may
	// be
	// different from the job location. The service attempts
	// to geolocate the provided address, and populates a more
	// specific
	// location wherever possible in DerivedInfo.headquarters_location.
	HeadquartersAddress string `json:"headquartersAddress,omitempty"`

	// HiringAgency: Optional.
	//
	// Set to true if it is the hiring agency that post jobs for
	// other
	// employers.
	//
	// Defaults to false if not provided.
	HiringAgency bool `json:"hiringAgency,omitempty"`

	// ImageUri: Optional.
	//
	// A URI that hosts the employer's company logo.
	ImageUri string `json:"imageUri,omitempty"`

	// KeywordSearchableJobCustomAttributes: Optional.
	//
	// A list of keys of filterable Job.custom_attributes,
	// whose
	// corresponding `string_values` are used in keyword search. Jobs
	// with
	// `string_values` under these specified field keys are returned if
	// any
	// of the values matches the search keyword. Custom field values
	// with
	// parenthesis, brackets and special symbols won't be properly
	// searchable,
	// and those keyword queries need to be surrounded by quotes.
	KeywordSearchableJobCustomAttributes []string `json:"keywordSearchableJobCustomAttributes,omitempty"`

	// Name: Required during company update.
	//
	// The resource name for a company. This is generated by the service
	// when a
	// company is created.
	//
	// The format is "projects/{project_id}/companies/{company_id}", for
	// example,
	// "projects/api-test-project/companies/foo".
	Name string `json:"name,omitempty"`

	// Size: Optional.
	//
	// The employer's company size.
	//
	// Possible values:
	//   "COMPANY_SIZE_UNSPECIFIED" - Default value if the size is not
	// specified.
	//   "MINI" - The company has less than 50 employees.
	//   "SMALL" - The company has between 50 and 99 employees.
	//   "SMEDIUM" - The company has between 100 and 499 employees.
	//   "MEDIUM" - The company has between 500 and 999 employees.
	//   "BIG" - The company has between 1,000 and 4,999 employees.
	//   "BIGGER" - The company has between 5,000 and 9,999 employees.
	//   "GIANT" - The company has 10,000 or more employees.
	Size string `json:"size,omitempty"`

	// Suspended: Output only. Indicates whether a company is flagged to be
	// suspended from
	// public availability by the service when job content appears
	// suspicious,
	// abusive, or spammy.
	Suspended bool `json:"suspended,omitempty"`

	// WebsiteUri: Optional.
	//
	// The URI representing the company's primary web site or home page,
	// for example, "https://www.google.com".
	//
	// The maximum number of allowed characters is 255.
	WebsiteUri string `json:"websiteUri,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "CareerSiteUri") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "CareerSiteUri") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *Company) MarshalJSON() ([]byte, error) {
	type NoMethod Company
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CompanyDerivedInfo: Derived details about the company.
type CompanyDerivedInfo struct {
	// HeadquartersLocation: A structured headquarters location of the
	// company, resolved from
	// Company.hq_location if provided.
	HeadquartersLocation *Location `json:"headquartersLocation,omitempty"`

	// ForceSendFields is a list of field names (e.g.
	// "HeadquartersLocation") to unconditionally include in API requests.
	// By default, fields with empty values are omitted from API requests.
	// However, any non-pointer, non-interface field appearing in
	// ForceSendFields will be sent to the server regardless of whether the
	// field is empty or not. This may be used to include empty fields in
	// Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "HeadquartersLocation") to
	// include in API requests with the JSON null value. By default, fields
	// with empty values are omitted from API requests. However, any field
	// with an empty value appearing in NullFields will be sent to the
	// server as null. It is an error if a field in this list has a
	// non-empty value. This may be used to include null fields in Patch
	// requests.
	NullFields []string `json:"-"`
}

func (s *CompanyDerivedInfo) MarshalJSON() ([]byte, error) {
	type NoMethod CompanyDerivedInfo
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CompensationEntry: A compensation entry that represents one component
// of compensation, such
// as base pay, bonus, or other compensation type.
//
// Annualization: One compensation entry can be annualized if
// - it contains valid amount or range.
// - and its expected_units_per_year is set or can be derived.
// Its annualized range is determined as (amount or range)
// times
// expected_units_per_year.
type CompensationEntry struct {
	// Amount: Optional.
	//
	// Compensation amount.
	Amount *Money `json:"amount,omitempty"`

	// Description: Optional.
	//
	// Compensation description.  For example, could
	// indicate equity terms or provide additional context to an
	// estimated
	// bonus.
	Description string `json:"description,omitempty"`

	// ExpectedUnitsPerYear: Optional.
	//
	// Expected number of units paid each year. If not specified,
	// when
	// Job.employment_types is FULLTIME, a default value is inferred
	// based on unit. Default values:
	// - HOURLY: 2080
	// - DAILY: 260
	// - WEEKLY: 52
	// - MONTHLY: 12
	// - ANNUAL: 1
	ExpectedUnitsPerYear float64 `json:"expectedUnitsPerYear,omitempty"`

	// Range: Optional.
	//
	// Compensation range.
	Range *CompensationRange `json:"range,omitempty"`

	// Type: Optional.
	//
	// Compensation type.
	//
	// Default is CompensationUnit.OTHER_COMPENSATION_TYPE.
	//
	// Possible values:
	//   "COMPENSATION_TYPE_UNSPECIFIED" - Default value.
	//   "BASE" - Base compensation: Refers to the fixed amount of money
	// paid to an
	// employee by an employer in return for work performed. Base
	// compensation
	// does not include benefits, bonuses or any other potential
	// compensation
	// from an employer.
	//   "BONUS" - Bonus.
	//   "SIGNING_BONUS" - Signing bonus.
	//   "EQUITY" - Equity.
	//   "PROFIT_SHARING" - Profit sharing.
	//   "COMMISSIONS" - Commission.
	//   "TIPS" - Tips.
	//   "OTHER_COMPENSATION_TYPE" - Other compensation type.
	Type string `json:"type,omitempty"`

	// Unit: Optional.
	//
	// Frequency of the specified amount.
	//
	// Default is CompensationUnit.OTHER_COMPENSATION_UNIT.
	//
	// Possible values:
	//   "COMPENSATION_UNIT_UNSPECIFIED" - Default value.
	//   "HOURLY" - Hourly.
	//   "DAILY" - Daily.
	//   "WEEKLY" - Weekly
	//   "MONTHLY" - Monthly.
	//   "YEARLY" - Yearly.
	//   "ONE_TIME" - One time.
	//   "OTHER_COMPENSATION_UNIT" - Other compensation units.
	Unit string `json:"unit,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Amount") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Amount") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *CompensationEntry) MarshalJSON() ([]byte, error) {
	type NoMethod CompensationEntry
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

func (s *CompensationEntry) UnmarshalJSON(data []byte) error {
	type NoMethod CompensationEntry
	var s1 struct {
		ExpectedUnitsPerYear gensupport.JSONFloat64 `json:"expectedUnitsPerYear"`
		*NoMethod
	}
	s1.NoMethod = (*NoMethod)(s)
	if err := json.Unmarshal(data, &s1); err != nil {
		return err
	}
	s.ExpectedUnitsPerYear = float64(s1.ExpectedUnitsPerYear)
	return nil
}

// CompensationFilter: Input only.
//
// Filter on job compensation type and amount.
type CompensationFilter struct {
	// IncludeJobsWithUnspecifiedCompensationRange: Optional.
	//
	// Whether to include jobs whose compensation range is unspecified.
	IncludeJobsWithUnspecifiedCompensationRange bool `json:"includeJobsWithUnspecifiedCompensationRange,omitempty"`

	// Range: Optional.
	//
	// Compensation range.
	Range *CompensationRange `json:"range,omitempty"`

	// Type: Required.
	//
	// Type of filter.
	//
	// Possible values:
	//   "FILTER_TYPE_UNSPECIFIED" - Filter type unspecified. Position
	// holder, INVALID, should never be used.
	//   "UNIT_ONLY" - Filter by `base compensation entry's` unit. A job is
	// a match if and
	// only if the job contains a base CompensationEntry and the
	// base
	// CompensationEntry's unit matches provided units.
	// Populate one or more units.
	//
	// See CompensationInfo.CompensationEntry for definition of
	// base compensation entry.
	//   "UNIT_AND_AMOUNT" - Filter by `base compensation entry's` unit and
	// amount / range. A job
	// is a match if and only if the job contains a base CompensationEntry,
	// and
	// the base entry's unit matches provided compensation_units and
	// amount
	// or range overlaps with provided compensation_range.
	//
	// See CompensationInfo.CompensationEntry for definition of
	// base compensation entry.
	//
	// Set exactly one units and populate range.
	//   "ANNUALIZED_BASE_AMOUNT" - Filter by annualized base compensation
	// amount and `base compensation
	// entry's` unit. Populate range and zero or more units.
	//   "ANNUALIZED_TOTAL_AMOUNT" - Filter by annualized total compensation
	// amount and `base compensation
	// entry's` unit . Populate range and zero or more units.
	Type string `json:"type,omitempty"`

	// Units: Required.
	//
	// Specify desired `base compensation
	// entry's`
	// CompensationInfo.CompensationUnit.
	//
	// Possible values:
	//   "COMPENSATION_UNIT_UNSPECIFIED" - Default value.
	//   "HOURLY" - Hourly.
	//   "DAILY" - Daily.
	//   "WEEKLY" - Weekly
	//   "MONTHLY" - Monthly.
	//   "YEARLY" - Yearly.
	//   "ONE_TIME" - One time.
	//   "OTHER_COMPENSATION_UNIT" - Other compensation units.
	Units []string `json:"units,omitempty"`

	// ForceSendFields is a list of field names (e.g.
	// "IncludeJobsWithUnspecifiedCompensationRange") to unconditionally
	// include in API requests. By default, fields with empty values are
	// omitted from API requests. However, any non-pointer, non-interface
	// field appearing in ForceSendFields will be sent to the server
	// regardless of whether the field is empty or not. This may be used to
	// include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g.
	// "IncludeJobsWithUnspecifiedCompensationRange") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *CompensationFilter) MarshalJSON() ([]byte, error) {
	type NoMethod CompensationFilter
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CompensationHistogramRequest: Input only.
//
// Compensation based histogram request.
type CompensationHistogramRequest struct {
	// BucketingOption: Required.
	//
	// Numeric histogram options, like buckets, whether include min or max
	// value.
	BucketingOption *NumericBucketingOption `json:"bucketingOption,omitempty"`

	// Type: Required.
	//
	// Type of the request, representing which field the histogramming
	// should be
	// performed over. A single request can only specify one histogram of
	// each
	// `CompensationHistogramRequestType`.
	//
	// Possible values:
	//   "COMPENSATION_HISTOGRAM_REQUEST_TYPE_UNSPECIFIED" - Default value.
	// Invalid.
	//   "BASE" - Histogram by job's base compensation. See
	// CompensationEntry for
	// definition of base compensation.
	//   "ANNUALIZED_BASE" - Histogram by job's annualized base
	// compensation. See CompensationEntry
	// for definition of annualized base compensation.
	//   "ANNUALIZED_TOTAL" - Histogram by job's annualized total
	// compensation. See CompensationEntry
	// for definition of annualized total compensation.
	Type string `json:"type,omitempty"`

	// ForceSendFields is a list of field names (e.g. "BucketingOption") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "BucketingOption") to
	// include in API requests with the JSON null value. By default, fields
	// with empty values are omitted from API requests. However, any field
	// with an empty value appearing in NullFields will be sent to the
	// server as null. It is an error if a field in this list has a
	// non-empty value. This may be used to include null fields in Patch
	// requests.
	NullFields []string `json:"-"`
}

func (s *CompensationHistogramRequest) MarshalJSON() ([]byte, error) {
	type NoMethod CompensationHistogramRequest
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CompensationHistogramResult: Output only.
//
// Compensation based histogram result.
type CompensationHistogramResult struct {
	// Result: Histogram result.
	Result *NumericBucketingResult `json:"result,omitempty"`

	// Type: Type of the request, corresponding
	// to
	// CompensationHistogramRequest.type.
	//
	// Possible values:
	//   "COMPENSATION_HISTOGRAM_REQUEST_TYPE_UNSPECIFIED" - Default value.
	// Invalid.
	//   "BASE" - Histogram by job's base compensation. See
	// CompensationEntry for
	// definition of base compensation.
	//   "ANNUALIZED_BASE" - Histogram by job's annualized base
	// compensation. See CompensationEntry
	// for definition of annualized base compensation.
	//   "ANNUALIZED_TOTAL" - Histogram by job's annualized total
	// compensation. See CompensationEntry
	// for definition of annualized total compensation.
	Type string `json:"type,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Result") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Result") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *CompensationHistogramResult) MarshalJSON() ([]byte, error) {
	type NoMethod CompensationHistogramResult
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CompensationInfo: Job compensation details.
type CompensationInfo struct {
	// AnnualizedBaseCompensationRange: Output only. Annualized base
	// compensation range. Computed as
	// base compensation entry's CompensationEntry.compensation
	// times
	// CompensationEntry.expected_units_per_year.
	//
	// See CompensationEntry for explanation on compensation annualization.
	AnnualizedBaseCompensationRange *CompensationRange `json:"annualizedBaseCompensationRange,omitempty"`

	// AnnualizedTotalCompensationRange: Output only. Annualized total
	// compensation range. Computed as
	// all compensation entries' CompensationEntry.compensation
	// times
	// CompensationEntry.expected_units_per_year.
	//
	// See CompensationEntry for explanation on compensation annualization.
	AnnualizedTotalCompensationRange *CompensationRange `json:"annualizedTotalCompensationRange,omitempty"`

	// Entries: Optional.
	//
	// Job compensation information.
	//
	// At most one entry can be of
	// type
	// CompensationInfo.CompensationType.BASE, which is
	// referred as ** base compensation entry ** for the job.
	Entries []*CompensationEntry `json:"entries,omitempty"`

	// ForceSendFields is a list of field names (e.g.
	// "AnnualizedBaseCompensationRange") to unconditionally include in API
	// requests. By default, fields with empty values are omitted from API
	// requests. However, any non-pointer, non-interface field appearing in
	// ForceSendFields will be sent to the server regardless of whether the
	// field is empty or not. This may be used to include empty fields in
	// Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g.
	// "AnnualizedBaseCompensationRange") to include in API requests with
	// the JSON null value. By default, fields with empty values are omitted
	// from API requests. However, any field with an empty value appearing
	// in NullFields will be sent to the server as null. It is an error if a
	// field in this list has a non-empty value. This may be used to include
	// null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *CompensationInfo) MarshalJSON() ([]byte, error) {
	type NoMethod CompensationInfo
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CompensationRange: Compensation range.
type CompensationRange struct {
	// MaxCompensation: Optional.
	//
	// The maximum amount of compensation. If left empty, the value is
	// set
	// to a maximal compensation value and the currency code is set to
	// match the currency code of
	// min_compensation.
	MaxCompensation *Money `json:"maxCompensation,omitempty"`

	// MinCompensation: Optional.
	//
	// The minimum amount of compensation. If left empty, the value is
	// set
	// to zero and the currency code is set to match the
	// currency code of max_compensation.
	MinCompensation *Money `json:"minCompensation,omitempty"`

	// ForceSendFields is a list of field names (e.g. "MaxCompensation") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "MaxCompensation") to
	// include in API requests with the JSON null value. By default, fields
	// with empty values are omitted from API requests. However, any field
	// with an empty value appearing in NullFields will be sent to the
	// server as null. It is an error if a field in this list has a
	// non-empty value. This may be used to include null fields in Patch
	// requests.
	NullFields []string `json:"-"`
}

func (s *CompensationRange) MarshalJSON() ([]byte, error) {
	type NoMethod CompensationRange
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CompleteQueryResponse: Output only.
//
// Response of auto-complete query.
type CompleteQueryResponse struct {
	// CompletionResults: Results of the matching job/company candidates.
	CompletionResults []*CompletionResult `json:"completionResults,omitempty"`

	// Metadata: Additional information for the API invocation, such as the
	// request
	// tracking id.
	Metadata *ResponseMetadata `json:"metadata,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "CompletionResults")
	// to unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "CompletionResults") to
	// include in API requests with the JSON null value. By default, fields
	// with empty values are omitted from API requests. However, any field
	// with an empty value appearing in NullFields will be sent to the
	// server as null. It is an error if a field in this list has a
	// non-empty value. This may be used to include null fields in Patch
	// requests.
	NullFields []string `json:"-"`
}

func (s *CompleteQueryResponse) MarshalJSON() ([]byte, error) {
	type NoMethod CompleteQueryResponse
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CompletionResult: Output only.
//
// Resource that represents completion results.
type CompletionResult struct {
	// ImageUri: The URI of the company image for
	// CompletionType.COMPANY_NAME.
	ImageUri string `json:"imageUri,omitempty"`

	// Suggestion: The suggestion for the query.
	Suggestion string `json:"suggestion,omitempty"`

	// Type: The completion topic.
	//
	// Possible values:
	//   "COMPLETION_TYPE_UNSPECIFIED" - Default value.
	//   "JOB_TITLE" - Only suggest job titles.
	//   "COMPANY_NAME" - Only suggest company names.
	//   "COMBINED" - Suggest both job titles and company names.
	Type string `json:"type,omitempty"`

	// ForceSendFields is a list of field names (e.g. "ImageUri") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "ImageUri") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *CompletionResult) MarshalJSON() ([]byte, error) {
	type NoMethod CompletionResult
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CreateCompanyRequest: Input only.
//
// The Request of the CreateCompany method.
type CreateCompanyRequest struct {
	// Company: Required.
	//
	// The company to be created.
	Company *Company `json:"company,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Company") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Company") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *CreateCompanyRequest) MarshalJSON() ([]byte, error) {
	type NoMethod CreateCompanyRequest
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CreateJobRequest: Input only.
//
// Create job request.
type CreateJobRequest struct {
	// Job: Required.
	//
	// The Job to be created.
	Job *Job `json:"job,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Job") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Job") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *CreateJobRequest) MarshalJSON() ([]byte, error) {
	type NoMethod CreateJobRequest
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CustomAttribute: Custom attribute values that are either filterable
// or non-filterable.
type CustomAttribute struct {
	// Filterable: Optional.
	//
	// If the `filterable` flag is true, custom field values are
	// searchable.
	// If false, values are not searchable.
	//
	// Default is false.
	Filterable bool `json:"filterable,omitempty"`

	// LongValues: Optional but exactly one of string_values or long_values
	// must
	// be specified.
	//
	// This field is used to perform number range search.
	// (`EQ`, `GT`, `GE`, `LE`, `LT`) over filterable
	// `long_value`.
	//
	// Currently at most 1 long_values is supported.
	LongValues googleapi.Int64s `json:"longValues,omitempty"`

	// StringValues: Optional but exactly one of string_values or
	// long_values must
	// be specified.
	//
	// This field is used to perform a string match (`CASE_SENSITIVE_MATCH`
	// or
	// `CASE_INSENSITIVE_MATCH`) search.
	// For filterable `string_value`s, a maximum total number of 200
	// values
	// is allowed, with each `string_value` has a byte size of no more
	// than
	// 255B. For unfilterable `string_values`, the maximum total byte size
	// of
	// unfilterable `string_values` is 50KB.
	//
	// Empty string is not allowed.
	StringValues []string `json:"stringValues,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Filterable") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Filterable") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *CustomAttribute) MarshalJSON() ([]byte, error) {
	type NoMethod CustomAttribute
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CustomAttributeHistogramRequest: Custom attributes histogram request.
// An error is thrown if neither
// string_value_histogram or long_value_histogram_bucketing_option
// has
// been defined.
type CustomAttributeHistogramRequest struct {
	// Key: Required.
	//
	// Specifies the custom field key to perform a histogram on. If
	// specified
	// without `long_value_histogram_bucketing_option`, histogram on string
	// values
	// of the given `key` is triggered, otherwise histogram is performed on
	// long
	// values.
	Key string `json:"key,omitempty"`

	// LongValueHistogramBucketingOption: Optional.
	//
	// Specifies buckets used to perform a range histogram on
	// Job's
	// filterable long custom field values, or min/max value requirements.
	LongValueHistogramBucketingOption *NumericBucketingOption `json:"longValueHistogramBucketingOption,omitempty"`

	// StringValueHistogram: Optional. If set to true, the response includes
	// the histogram value for
	// each key as a string.
	StringValueHistogram bool `json:"stringValueHistogram,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Key") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Key") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *CustomAttributeHistogramRequest) MarshalJSON() ([]byte, error) {
	type NoMethod CustomAttributeHistogramRequest
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CustomAttributeHistogramResult: Output only.
//
// Custom attribute histogram result.
type CustomAttributeHistogramResult struct {
	// Key: Stores the key of custom attribute the histogram is performed
	// on.
	Key string `json:"key,omitempty"`

	// LongValueHistogramResult: Stores bucketed histogram counting result
	// or min/max values for
	// custom attribute long values associated with `key`.
	LongValueHistogramResult *NumericBucketingResult `json:"longValueHistogramResult,omitempty"`

	// StringValueHistogramResult: Stores a map from the values of string
	// custom field associated
	// with `key` to the number of jobs with that value in this histogram
	// result.
	StringValueHistogramResult map[string]int64 `json:"stringValueHistogramResult,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Key") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Key") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *CustomAttributeHistogramResult) MarshalJSON() ([]byte, error) {
	type NoMethod CustomAttributeHistogramResult
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// DeviceInfo: Device information collected from the job seeker,
// candidate, or
// other entity conducting the job search. Providing this information
// improves
// the quality of the search results across devices.
type DeviceInfo struct {
	// DeviceType: Optional.
	//
	// Type of the device.
	//
	// Possible values:
	//   "DEVICE_TYPE_UNSPECIFIED" - The device type isn't specified.
	//   "WEB" - A desktop web browser, such as, Chrome, Firefox, Safari, or
	// Internet
	// Explorer)
	//   "MOBILE_WEB" - A mobile device web browser, such as a phone or
	// tablet with a Chrome
	// browser.
	//   "ANDROID" - An Android device native application.
	//   "IOS" - An iOS device native application.
	//   "BOT" - A bot, as opposed to a device operated by human beings,
	// such as a web
	// crawler.
	//   "OTHER" - Other devices types.
	DeviceType string `json:"deviceType,omitempty"`

	// Id: Optional.
	//
	// A device-specific ID. The ID must be a unique identifier
	// that
	// distinguishes the device from other devices.
	Id string `json:"id,omitempty"`

	// ForceSendFields is a list of field names (e.g. "DeviceType") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "DeviceType") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *DeviceInfo) MarshalJSON() ([]byte, error) {
	type NoMethod DeviceInfo
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// Empty: A generic empty message that you can re-use to avoid defining
// duplicated
// empty messages in your APIs. A typical example is to use it as the
// request
// or the response type of an API method. For instance:
//
//     service Foo {
//       rpc Bar(google.protobuf.Empty) returns
// (google.protobuf.Empty);
//     }
//
// The JSON representation for `Empty` is empty JSON object `{}`.
type Empty struct {
	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`
}

// HistogramFacets: Input only.
//
// Histogram facets to be specified in SearchJobsRequest.
type HistogramFacets struct {
	// CompensationHistogramFacets: Optional.
	//
	// Specifies compensation field-based histogram requests.
	// Duplicate values of CompensationHistogramRequest.type are not
	// allowed.
	CompensationHistogramFacets []*CompensationHistogramRequest `json:"compensationHistogramFacets,omitempty"`

	// CustomAttributeHistogramFacets: Optional.
	//
	// Specifies the custom attributes histogram requests.
	// Duplicate values of CustomAttributeHistogramRequest.key are
	// not
	// allowed.
	CustomAttributeHistogramFacets []*CustomAttributeHistogramRequest `json:"customAttributeHistogramFacets,omitempty"`

	// SimpleHistogramFacets: Optional.
	//
	// Specifies the simple type of histogram facets, for
	// example,
	// `COMPANY_SIZE`, `EMPLOYMENT_TYPE` etc.
	//
	// Possible values:
	//   "SEARCH_TYPE_UNSPECIFIED" - The default value if search type is not
	// specified.
	//   "COMPANY_ID" - Filter by the company id field.
	//   "EMPLOYMENT_TYPE" - Filter by the employment type field, such as
	// `FULL_TIME` or `PART_TIME`.
	//   "COMPANY_SIZE" - Filter by the company size type field, such as
	// `BIG`, `SMALL` or `BIGGER`.
	//   "DATE_PUBLISHED" - Filter by the date published field. Possible
	// return values are:
	// * PAST_24_HOURS (The past 24 hours)
	// * PAST_3_DAYS (The past 3 days)
	// * PAST_WEEK (The past 7 days)
	// * PAST_MONTH (The past 30 days)
	// * PAST_YEAR (The past 365 days)
	//   "EDUCATION_LEVEL" - Filter by the required education level of the
	// job.
	//   "EXPERIENCE_LEVEL" - Filter by the required experience level of the
	// job.
	//   "ADMIN_1" - Filter by Admin1, which is a global placeholder
	// for
	// referring to state, province, or the particular term a country uses
	// to
	// define the geographic structure below the country level.
	// Examples include states codes such as "CA", "IL", "NY",
	// and
	// provinces, such as "BC".
	//   "COUNTRY" - Filter by the country code of job, such as US, JP, FR.
	//   "CITY" - Filter by the "city name", "Admin1 code", for
	// example,
	// "Mountain View, CA" or "New York, NY".
	//   "LOCALE" - Filter by the locale field of a job, such as "en-US",
	// "fr-FR".
	//
	// This is the BCP-47 language code, such as "en-US" or "sr-Latn".
	// For more information, see
	// [Tags for Identifying Languages](https://tools.ietf.org/html/bcp47).
	//   "LANGUAGE" - Filter by the language code portion of the locale
	// field, such as "en" or
	// "fr".
	//   "CATEGORY" - Filter by the Category.
	//   "CITY_COORDINATE" - Filter by the city center GPS coordinate
	// (latitude and longitude), for
	// example, 37.4038522,-122.0987765. Since the coordinates of a city
	// center
	// can change, clients may need to refresh them periodically.
	//   "ADMIN_1_COUNTRY" - A combination of state or province code with a
	// country code. This field
	// differs from `JOB_ADMIN1`, which can be used in multiple countries.
	//   "COMPANY_DISPLAY_NAME" - Company display name.
	//   "BASE_COMPENSATION_UNIT" - Base compensation unit.
	SimpleHistogramFacets []string `json:"simpleHistogramFacets,omitempty"`

	// ForceSendFields is a list of field names (e.g.
	// "CompensationHistogramFacets") to unconditionally include in API
	// requests. By default, fields with empty values are omitted from API
	// requests. However, any non-pointer, non-interface field appearing in
	// ForceSendFields will be sent to the server regardless of whether the
	// field is empty or not. This may be used to include empty fields in
	// Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g.
	// "CompensationHistogramFacets") to include in API requests with the
	// JSON null value. By default, fields with empty values are omitted
	// from API requests. However, any field with an empty value appearing
	// in NullFields will be sent to the server as null. It is an error if a
	// field in this list has a non-empty value. This may be used to include
	// null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *HistogramFacets) MarshalJSON() ([]byte, error) {
	type NoMethod HistogramFacets
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// HistogramResult: Output only.
//
// Result of a histogram call. The response contains the histogram map
// for the
// search type specified by HistogramResult.field.
// The response is a map of each filter value to the corresponding count
// of
// jobs for that filter.
type HistogramResult struct {
	// SearchType: The Histogram search filters.
	//
	// Possible values:
	//   "SEARCH_TYPE_UNSPECIFIED" - The default value if search type is not
	// specified.
	//   "COMPANY_ID" - Filter by the company id field.
	//   "EMPLOYMENT_TYPE" - Filter by the employment type field, such as
	// `FULL_TIME` or `PART_TIME`.
	//   "COMPANY_SIZE" - Filter by the company size type field, such as
	// `BIG`, `SMALL` or `BIGGER`.
	//   "DATE_PUBLISHED" - Filter by the date published field. Possible
	// return values are:
	// * PAST_24_HOURS (The past 24 hours)
	// * PAST_3_DAYS (The past 3 days)
	// * PAST_WEEK (The past 7 days)
	// * PAST_MONTH (The past 30 days)
	// * PAST_YEAR (The past 365 days)
	//   "EDUCATION_LEVEL" - Filter by the required education level of the
	// job.
	//   "EXPERIENCE_LEVEL" - Filter by the required experience level of the
	// job.
	//   "ADMIN_1" - Filter by Admin1, which is a global placeholder
	// for
	// referring to state, province, or the particular term a country uses
	// to
	// define the geographic structure below the country level.
	// Examples include states codes such as "CA", "IL", "NY",
	// and
	// provinces, such as "BC".
	//   "COUNTRY" - Filter by the country code of job, such as US, JP, FR.
	//   "CITY" - Filter by the "city name", "Admin1 code", for
	// example,
	// "Mountain View, CA" or "New York, NY".
	//   "LOCALE" - Filter by the locale field of a job, such as "en-US",
	// "fr-FR".
	//
	// This is the BCP-47 language code, such as "en-US" or "sr-Latn".
	// For more information, see
	// [Tags for Identifying Languages](https://tools.ietf.org/html/bcp47).
	//   "LANGUAGE" - Filter by the language code portion of the locale
	// field, such as "en" or
	// "fr".
	//   "CATEGORY" - Filter by the Category.
	//   "CITY_COORDINATE" - Filter by the city center GPS coordinate
	// (latitude and longitude), for
	// example, 37.4038522,-122.0987765. Since the coordinates of a city
	// center
	// can change, clients may need to refresh them periodically.
	//   "ADMIN_1_COUNTRY" - A combination of state or province code with a
	// country code. This field
	// differs from `JOB_ADMIN1`, which can be used in multiple countries.
	//   "COMPANY_DISPLAY_NAME" - Company display name.
	//   "BASE_COMPENSATION_UNIT" - Base compensation unit.
	SearchType string `json:"searchType,omitempty"`

	// Values: A map from the values of field to the number of jobs with
	// that value
	// in this search result.
	//
	// Key: search type (filter names, such as the companyName).
	//
	// Values: the count of jobs that match the filter for this search.
	Values map[string]int64 `json:"values,omitempty"`

	// ForceSendFields is a list of field names (e.g. "SearchType") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "SearchType") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *HistogramResult) MarshalJSON() ([]byte, error) {
	type NoMethod HistogramResult
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// HistogramResults: Output only.
//
// Histogram results that match HistogramFacets specified
// in
// SearchJobsRequest.
type HistogramResults struct {
	// CompensationHistogramResults: Specifies compensation field-based
	// histogram results that
	// match
	// HistogramFacets.compensation_histogram_requests.
	CompensationHistogramResults []*CompensationHistogramResult `json:"compensationHistogramResults,omitempty"`

	// CustomAttributeHistogramResults: Specifies histogram results for
	// custom attributes that
	// match
	// HistogramFacets.custom_attribute_histogram_facets.
	CustomAttributeHistogramResults []*CustomAttributeHistogramResult `json:"customAttributeHistogramResults,omitempty"`

	// SimpleHistogramResults: Specifies histogram results that
	// matches
	// HistogramFacets.simple_histogram_facets.
	SimpleHistogramResults []*HistogramResult `json:"simpleHistogramResults,omitempty"`

	// ForceSendFields is a list of field names (e.g.
	// "CompensationHistogramResults") to unconditionally include in API
	// requests. By default, fields with empty values are omitted from API
	// requests. However, any non-pointer, non-interface field appearing in
	// ForceSendFields will be sent to the server regardless of whether the
	// field is empty or not. This may be used to include empty fields in
	// Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g.
	// "CompensationHistogramResults") to include in API requests with the
	// JSON null value. By default, fields with empty values are omitted
	// from API requests. However, any field with an empty value appearing
	// in NullFields will be sent to the server as null. It is an error if a
	// field in this list has a non-empty value. This may be used to include
	// null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *HistogramResults) MarshalJSON() ([]byte, error) {
	type NoMethod HistogramResults
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// Job: A Job resource represents a job posting (also referred to as a
// "job listing"
// or "job requisition"). A job belongs to a Company, which is the
// hiring
// entity responsible for the job.
type Job struct {
	// Addresses: Optional but strongly recommended for the best service
	// experience.
	//
	// Location(s) where the employer is looking to hire for this job
	// posting.
	//
	// Specifying the full street address(es) of the hiring location
	// enables
	// better API results, especially job searches by commute time.
	//
	// At most 50 locations are allowed for best search performance. If a
	// job has
	// more locations, it is suggested to split it into multiple jobs with
	// unique
	// requisition_ids (e.g. 'ReqA' becomes 'ReqA-1', 'ReqA-2', etc.)
	// as
	// multiple jobs with the same company_name, language_code
	// and
	// requisition_id are not allowed. If the original requisition_id
	// must
	// be preserved, a custom field should be used for storage. It is
	// also
	// suggested to group the locations that close to each other in the same
	// job
	// for better search experience.
	//
	// The maximum number of allowed characters is 500.
	Addresses []string `json:"addresses,omitempty"`

	// ApplicationInfo: Required. At least one field within ApplicationInfo
	// must be specified.
	//
	// Job application information.
	ApplicationInfo *ApplicationInfo `json:"applicationInfo,omitempty"`

	// CompanyDisplayName: Output only. Display name of the company listing
	// the job.
	CompanyDisplayName string `json:"companyDisplayName,omitempty"`

	// CompanyName: Required.
	//
	// The resource name of the company listing the job, such
	// as
	// "projects/api-test-project/companies/foo".
	CompanyName string `json:"companyName,omitempty"`

	// CompensationInfo: Optional.
	//
	// Job compensation information.
	CompensationInfo *CompensationInfo `json:"compensationInfo,omitempty"`

	// CustomAttributes: Optional.
	//
	// A map of fields to hold both filterable and non-filterable custom
	// job
	// attributes that are not covered by the provided structured
	// fields.
	//
	// The keys of the map are strings up to 64 bytes and must match
	// the
	// pattern: a-zA-Z*. For example, key0LikeThis or
	// KEY_1_LIKE_THIS.
	//
	// At most 100 filterable and at most 100 unfilterable keys are
	// supported.
	// For filterable `string_values`, across all keys at most 200 values
	// are
	// allowed, with each string no more than 255 characters. For
	// unfilterable
	// `string_values`, the maximum total size of `string_values` across all
	// keys
	// is 50KB.
	CustomAttributes map[string]CustomAttribute `json:"customAttributes,omitempty"`

	// DegreeTypes: Optional.
	//
	// The desired education degrees for the job, such as Bachelors,
	// Masters.
	//
	// Possible values:
	//   "DEGREE_TYPE_UNSPECIFIED" - Default value. Represents no degree, or
	// early childhood education.
	// Maps to ISCED code 0.
	// Ex) Kindergarten
	//   "PRIMARY_EDUCATION" - Primary education which is typically the
	// first stage of compulsory
	// education. ISCED code 1.
	// Ex) Elementary school
	//   "LOWER_SECONDARY_EDUCATION" - Lower secondary education; First
	// stage of secondary education building on
	// primary education, typically with a more subject-oriented
	// curriculum.
	// ISCED code 2.
	// Ex) Middle school
	//   "UPPER_SECONDARY_EDUCATION" - Middle education; Second/final stage
	// of secondary education preparing for
	// tertiary education and/or providing skills relevant to
	// employment.
	// Usually with an increased range of subject options and streams.
	// ISCED
	// code 3.
	// Ex) High school
	//   "ADULT_REMEDIAL_EDUCATION" - Adult Remedial Education; Programmes
	// providing learning experiences that
	// build on secondary education and prepare for labour market entry
	// and/or
	// tertiary education. The content is broader than secondary but not
	// as
	// complex as tertiary education. ISCED code 4.
	//   "ASSOCIATES_OR_EQUIVALENT" - Associate's or equivalent; Short first
	// tertiary programmes that are
	// typically practically-based, occupationally-specific and prepare
	// for
	// labour market entry. These programmes may also provide a pathway to
	// other
	// tertiary programmes. ISCED code 5.
	//   "BACHELORS_OR_EQUIVALENT" - Bachelor's or equivalent; Programmes
	// designed to provide intermediate
	// academic and/or professional knowledge, skills and competencies
	// leading
	// to a first tertiary degree or equivalent qualification. ISCED code 6.
	//   "MASTERS_OR_EQUIVALENT" - Master's or equivalent; Programmes
	// designed to provide advanced academic
	// and/or professional knowledge, skills and competencies leading to
	// a
	// second tertiary degree or equivalent qualification. ISCED code 7.
	//   "DOCTORAL_OR_EQUIVALENT" - Doctoral or equivalent; Programmes
	// designed primarily to lead to an
	// advanced research qualification, usually concluding with the
	// submission
	// and defense of a substantive dissertation of publishable quality
	// based on
	// original research. ISCED code 8.
	DegreeTypes []string `json:"degreeTypes,omitempty"`

	// Department: Optional.
	//
	// The department or functional area within the company with the
	// open
	// position.
	//
	// The maximum number of allowed characters is 255.
	Department string `json:"department,omitempty"`

	// DerivedInfo: Output only. Derived details about the job posting.
	DerivedInfo *JobDerivedInfo `json:"derivedInfo,omitempty"`

	// Description: Required.
	//
	// The description of the job, which typically includes a
	// multi-paragraph
	// description of the company and related information. Separate fields
	// are
	// provided on the job object for responsibilities,
	// qualifications, and other job characteristics. Use of
	// these separate job fields is recommended.
	//
	// This field accepts and sanitizes HTML input, and also accepts
	// bold, italic, ordered list, and unordered list markup tags.
	//
	// The maximum number of allowed characters is 100,000.
	Description string `json:"description,omitempty"`

	// EmploymentTypes: Optional.
	//
	// The employment type(s) of a job, for example,
	// full time or
	// part time.
	//
	// Possible values:
	//   "EMPLOYMENT_TYPE_UNSPECIFIED" - The default value if the employment
	// type is not specified.
	//   "FULL_TIME" - The job requires working a number of hours that
	// constitute full
	// time employment, typically 40 or more hours per week.
	//   "PART_TIME" - The job entails working fewer hours than a full time
	// job,
	// typically less than 40 hours a week.
	//   "CONTRACTOR" - The job is offered as a contracted, as opposed to a
	// salaried employee,
	// position.
	//   "CONTRACT_TO_HIRE" - The job is offered as a contracted position
	// with the understanding
	// that it's converted into a full-time position at the end of
	// the
	// contract. Jobs of this type are also returned by a search
	// for
	// EmploymentType.CONTRACTOR jobs.
	//   "TEMPORARY" - The job is offered as a temporary employment
	// opportunity, usually
	// a short-term engagement.
	//   "INTERN" - The job is a fixed-term opportunity for students or
	// entry-level job
	// seekers to obtain on-the-job training, typically offered as a
	// summer
	// position.
	//   "VOLUNTEER" - The is an opportunity for an individual to volunteer,
	// where there's no
	// expectation of compensation for the provided services.
	//   "PER_DIEM" - The job requires an employee to work on an as-needed
	// basis with a
	// flexible schedule.
	//   "FLY_IN_FLY_OUT" - The job involves employing people in remote
	// areas and flying them
	// temporarily to the work site instead of relocating employees and
	// their
	// families permanently.
	//   "OTHER_EMPLOYMENT_TYPE" - The job does not fit any of the other
	// listed types.
	EmploymentTypes []string `json:"employmentTypes,omitempty"`

	// Incentives: Optional.
	//
	// A description of bonus, commission, and other compensation
	// incentives associated with the job not including salary or pay.
	//
	// The maximum number of allowed characters is 10,000.
	Incentives string `json:"incentives,omitempty"`

	// JobBenefits: Optional.
	//
	// The benefits included with the job.
	//
	// Possible values:
	//   "JOB_BENEFIT_UNSPECIFIED" - Default value if the type is not
	// specified.
	//   "CHILD_CARE" - The job includes access to programs that support
	// child care, such
	// as daycare.
	//   "DENTAL" - The job includes dental services covered by a
	// dental
	// insurance plan.
	//   "DOMESTIC_PARTNER" - The job offers specific benefits to domestic
	// partners.
	//   "FLEXIBLE_HOURS" - The job allows for a flexible work schedule.
	//   "MEDICAL" - The job includes health services covered by a medical
	// insurance plan.
	//   "LIFE_INSURANCE" - The job includes a life insurance plan provided
	// by the employer or
	// available for purchase by the employee.
	//   "PARENTAL_LEAVE" - The job allows for a leave of absence to a
	// parent to care for a newborn
	// child.
	//   "RETIREMENT_PLAN" - The job includes a workplace retirement plan
	// provided by the
	// employer or available for purchase by the employee.
	//   "SICK_DAYS" - The job allows for paid time off due to illness.
	//   "VACATION" - The job includes paid time off for vacation.
	//   "VISION" - The job includes vision services covered by a
	// vision
	// insurance plan.
	JobBenefits []string `json:"jobBenefits,omitempty"`

	// JobEndTime: Optional.
	//
	// The end timestamp of the job. Typically this field is used for
	// contracting
	// engagements. Invalid timestamps are ignored.
	JobEndTime string `json:"jobEndTime,omitempty"`

	// JobLevel: Optional.
	//
	// The experience level associated with the job, such as "Entry Level".
	//
	// Possible values:
	//   "JOB_LEVEL_UNSPECIFIED" - The default value if the level is not
	// specified.
	//   "ENTRY_LEVEL" - Entry-level individual contributors, typically with
	// less than 2 years of
	// experience in a similar role. Includes interns.
	//   "EXPERIENCED" - Experienced individual contributors, typically with
	// 2+ years of
	// experience in a similar role.
	//   "MANAGER" - Entry- to mid-level managers responsible for managing a
	// team of people.
	//   "DIRECTOR" - Senior-level managers responsible for managing teams
	// of managers.
	//   "EXECUTIVE" - Executive-level managers and above, including C-level
	// positions.
	JobLevel string `json:"jobLevel,omitempty"`

	// JobStartTime: Optional.
	//
	// The start timestamp of the job in UTC time zone. Typically this
	// field
	// is used for contracting engagements. Invalid timestamps are ignored.
	JobStartTime string `json:"jobStartTime,omitempty"`

	// LanguageCode: Optional.
	//
	// The language of the posting. This field is distinct from
	// any requirements for fluency that are associated with the
	// job.
	//
	// Language codes must be in BCP-47 format, such as "en-US" or
	// "sr-Latn".
	// For more information, see
	// [Tags for Identifying
	// Languages](https://tools.ietf.org/html/bcp47){:
	// class="external" target="_blank" }.
	//
	// If this field is unspecified and Job.description is present,
	// detected
	// language code based on Job.description is assigned,
	// otherwise
	// defaults to 'en_US'.
	LanguageCode string `json:"languageCode,omitempty"`

	// Name: Required during job update.
	//
	// The resource name for the job. This is generated by the service when
	// a
	// job is created.
	//
	// The format is "projects/{project_id}/jobs/{job_id}",
	// for example, "projects/api-test-project/jobs/1234".
	//
	// Use of this field in job queries and API calls is preferred over the
	// use of
	// requisition_id since this value is unique.
	Name string `json:"name,omitempty"`

	// PostingCreateTime: Output only. The timestamp when this job posting
	// was created.
	PostingCreateTime string `json:"postingCreateTime,omitempty"`

	// PostingExpireTime: Optional but strongly recommended for the best
	// service
	// experience.
	//
	// The expiration timestamp of the job. After this timestamp, the
	// job is marked as expired, and it no longer appears in search results.
	// The
	// expired job can't be deleted or listed by the DeleteJob and
	// ListJobs APIs, but it can be retrieved with the GetJob API or
	// updated with the UpdateJob API. An expired job can be updated
	// and
	// opened again by using a future expiration timestamp. Updating an
	// expired
	// job fails if there is another existing open job with same
	// company_name,
	// language_code and requisition_id.
	//
	// The expired jobs are retained in our system for 90 days. However,
	// the
	// overall expired job count cannot exceed 3 times the maximum of open
	// jobs
	// count over the past week, otherwise jobs with earlier expire time
	// are
	// cleaned first. Expired jobs are no longer accessible after they are
	// cleaned
	// out.
	//
	// Invalid timestamps are ignored, and treated as expire time not
	// provided.
	//
	// Timestamp before the instant request is made is considered valid, the
	// job
	// will be treated as expired immediately.
	//
	// If this value is not provided at the time of job creation or is
	// invalid,
	// the job posting expires after 30 days from the job's creation time.
	// For
	// example, if the job was created on 2017/01/01 13:00AM UTC with
	// an
	// unspecified expiration date, the job expires after 2017/01/31 13:00AM
	// UTC.
	//
	// If this value is not provided on job update, it depends on the field
	// masks
	// set by UpdateJobRequest.update_mask. If the field masks
	// include
	// expiry_time, or the masks are empty meaning that every field
	// is
	// updated, the job posting expires after 30 days from the job's
	// last
	// update time. Otherwise the expiration date isn't updated.
	PostingExpireTime string `json:"postingExpireTime,omitempty"`

	// PostingPublishTime: Optional.
	//
	// The timestamp this job posting was most recently published. The
	// default
	// value is the time the request arrives at the server. Invalid
	// timestamps are
	// ignored.
	PostingPublishTime string `json:"postingPublishTime,omitempty"`

	// PostingRegion: Optional.
	//
	// The job PostingRegion (for example, state, country) throughout
	// which
	// the job is available. If this field is set, a
	// LocationFilter in a search query within the job region
	// finds this job posting if an exact location match is not
	// specified.
	// If this field is set to PostingRegion.NATION_WIDE
	// or
	// [PostingRegion.ADMINISTRATIVE_AREA], setting job addresses
	// to the same location level as this field is strongly recommended.
	//
	// Possible values:
	//   "POSTING_REGION_UNSPECIFIED" - If the region is unspecified, the
	// job is only returned if it
	// matches the LocationFilter.
	//   "ADMINISTRATIVE_AREA" - In addition to exact location matching, job
	// posting is returned when the
	// LocationFilter in the search query is in the same administrative
	// area
	// as the returned job posting. For example, if a `ADMINISTRATIVE_AREA`
	// job
	// is posted in "CA, USA", it's returned if LocationFilter has
	// "Mountain View".
	//
	// Administrative area refers to top-level administrative subdivision of
	// this
	// country. For example, US state, IT region, UK constituent nation
	// and
	// JP prefecture.
	//   "NATION" - In addition to exact location matching, job is returned
	// when
	// LocationFilter in search query is in the same country as this
	// job.
	// For example, if a `NATION_WIDE` job is posted in "USA", it's
	// returned if LocationFilter has 'Mountain View'.
	//   "TELECOMMUTE" - Job allows employees to work remotely
	// (telecommute).
	// If locations are provided with this value, the job is
	// considered as having a location, but telecommuting is allowed.
	PostingRegion string `json:"postingRegion,omitempty"`

	// PostingUpdateTime: Output only. The timestamp when this job posting
	// was last updated.
	PostingUpdateTime string `json:"postingUpdateTime,omitempty"`

	// ProcessingOptions: Optional.
	//
	// Options for job processing.
	ProcessingOptions *ProcessingOptions `json:"processingOptions,omitempty"`

	// PromotionValue: Optional.
	//
	// A promotion value of the job, as determined by the client.
	// The value determines the sort order of the jobs returned when
	// searching for
	// jobs using the featured jobs search call, with higher promotional
	// values
	// being returned first and ties being resolved by relevance sort. Only
	// the
	// jobs with a promotionValue >0 are returned in a
	// FEATURED_JOB_SEARCH.
	//
	// Default value is 0, and negative values are treated as 0.
	PromotionValue int64 `json:"promotionValue,omitempty"`

	// Qualifications: Optional.
	//
	// A description of the qualifications required to perform the
	// job. The use of this field is recommended
	// as an alternative to using the more general description field.
	//
	// This field accepts and sanitizes HTML input, and also accepts
	// bold, italic, ordered list, and unordered list markup tags.
	//
	// The maximum number of allowed characters is 10,000.
	Qualifications string `json:"qualifications,omitempty"`

	// RequisitionId: Required.
	//
	// The requisition ID, also referred to as the posting ID, assigned by
	// the
	// client to identify a job. This field is intended to be used by
	// clients
	// for client identification and tracking of postings. A job is not
	// allowed
	// to be created if there is another job with the same
	// [company_name],
	// language_code and requisition_id.
	//
	// The maximum number of allowed characters is 255.
	RequisitionId string `json:"requisitionId,omitempty"`

	// Responsibilities: Optional.
	//
	// A description of job responsibilities. The use of this field
	// is
	// recommended as an alternative to using the more general
	// description
	// field.
	//
	// This field accepts and sanitizes HTML input, and also accepts
	// bold, italic, ordered list, and unordered list markup tags.
	//
	// The maximum number of allowed characters is 10,000.
	Responsibilities string `json:"responsibilities,omitempty"`

	// Title: Required.
	//
	// The title of the job, such as "Software Engineer"
	//
	// The maximum number of allowed characters is 500.
	Title string `json:"title,omitempty"`

	// Visibility: Optional.
	//
	// The visibility of the job.
	//
	// Defaults to Visibility.ACCOUNT_ONLY if not specified.
	//
	// Possible values:
	//   "VISIBILITY_UNSPECIFIED" - Default value.
	//   "ACCOUNT_ONLY" - The resource is only visible to the GCP account
	// who owns it.
	//   "SHARED_WITH_GOOGLE" - The resource is visible to the owner and may
	// be visible to other
	// applications and processes at Google.
	//   "SHARED_WITH_PUBLIC" - The resource is visible to the owner and may
	// be visible to all other API
	// clients.
	Visibility string `json:"visibility,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "Addresses") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Addresses") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *Job) MarshalJSON() ([]byte, error) {
	type NoMethod Job
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// JobDerivedInfo: Output only.
//
// Derived details about the job posting.
type JobDerivedInfo struct {
	// JobCategories: Job categories derived from Job.title and
	// Job.description.
	//
	// Possible values:
	//   "JOB_CATEGORY_UNSPECIFIED" - The default value if the category
	// isn't specified.
	//   "ACCOUNTING_AND_FINANCE" - An accounting and finance job, such as
	// an Accountant.
	//   "ADMINISTRATIVE_AND_OFFICE" - An administrative and office job,
	// such as an Administrative Assistant.
	//   "ADVERTISING_AND_MARKETING" - An advertising and marketing job,
	// such as Marketing Manager.
	//   "ANIMAL_CARE" - An animal care job, such as Veterinarian.
	//   "ART_FASHION_AND_DESIGN" - An art, fashion, or design job, such as
	// Designer.
	//   "BUSINESS_OPERATIONS" - A business operations job, such as Business
	// Operations Manager.
	//   "CLEANING_AND_FACILITIES" - A cleaning and facilities job, such as
	// Custodial Staff.
	//   "COMPUTER_AND_IT" - A computer and IT job, such as Systems
	// Administrator.
	//   "CONSTRUCTION" - A construction job, such as General Laborer.
	//   "CUSTOMER_SERVICE" - A customer service job, such s Cashier.
	//   "EDUCATION" - An education job, such as School Teacher.
	//   "ENTERTAINMENT_AND_TRAVEL" - An entertainment and travel job, such
	// as Flight Attendant.
	//   "FARMING_AND_OUTDOORS" - A farming or outdoor job, such as Park
	// Ranger.
	//   "HEALTHCARE" - A healthcare job, such as Registered Nurse.
	//   "HUMAN_RESOURCES" - A human resources job, such as Human Resources
	// Director.
	//   "INSTALLATION_MAINTENANCE_AND_REPAIR" - An installation,
	// maintenance, or repair job, such as Electrician.
	//   "LEGAL" - A legal job, such as Law Clerk.
	//   "MANAGEMENT" - A management job, often used in conjunction with
	// another category,
	// such as Store Manager.
	//   "MANUFACTURING_AND_WAREHOUSE" - A manufacturing or warehouse job,
	// such as Assembly Technician.
	//   "MEDIA_COMMUNICATIONS_AND_WRITING" - A media, communications, or
	// writing job, such as Media Relations.
	//   "OIL_GAS_AND_MINING" - An oil, gas or mining job, such as Offshore
	// Driller.
	//   "PERSONAL_CARE_AND_SERVICES" - A personal care and services job,
	// such as Hair Stylist.
	//   "PROTECTIVE_SERVICES" - A protective services job, such as Security
	// Guard.
	//   "REAL_ESTATE" - A real estate job, such as Buyer's Agent.
	//   "RESTAURANT_AND_HOSPITALITY" - A restaurant and hospitality job,
	// such as Restaurant Server.
	//   "SALES_AND_RETAIL" - A sales and/or retail job, such Sales
	// Associate.
	//   "SCIENCE_AND_ENGINEERING" - A science and engineering job, such as
	// Lab Technician.
	//   "SOCIAL_SERVICES_AND_NON_PROFIT" - A social services or non-profit
	// job, such as Case Worker.
	//   "SPORTS_FITNESS_AND_RECREATION" - A sports, fitness, or recreation
	// job, such as Personal Trainer.
	//   "TRANSPORTATION_AND_LOGISTICS" - A transportation or logistics job,
	// such as Truck Driver.
	JobCategories []string `json:"jobCategories,omitempty"`

	// Locations: Structured locations of the job, resolved from
	// Job.addresses.
	//
	// locations are exactly matched to Job.addresses in the same
	// order.
	Locations []*Location `json:"locations,omitempty"`

	// ForceSendFields is a list of field names (e.g. "JobCategories") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "JobCategories") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *JobDerivedInfo) MarshalJSON() ([]byte, error) {
	type NoMethod JobDerivedInfo
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// JobQuery: Input only.
//
// The query required to perform a search query.
type JobQuery struct {
	// CommuteFilter: Optional.
	//
	//  Allows filtering jobs by commute time with different travel methods
	// (for
	//  example, driving or public transit). Note: This only works with
	// COMMUTE
	//  MODE. When specified, [JobQuery.location_filters] is
	//  ignored.
	//
	//  Currently we don't support sorting by commute time.
	CommuteFilter *CommuteFilter `json:"commuteFilter,omitempty"`

	// CompanyDisplayNames: Optional.
	//
	// This filter specifies the exact company display
	// name of the jobs to search against.
	//
	// If a value isn't specified, jobs within the search results
	// are
	// associated with any company.
	//
	// If multiple values are specified, jobs within the search results may
	// be
	// associated with any of the specified companies.
	//
	// At most 20 company display name filters are allowed.
	CompanyDisplayNames []string `json:"companyDisplayNames,omitempty"`

	// CompanyNames: Optional.
	//
	// This filter specifies the company entities to search against.
	//
	// If a value isn't specified, jobs are searched for against
	// all
	// companies.
	//
	// If multiple values are specified, jobs are searched against
	// the
	// companies specified.
	//
	// The format is "projects/{project_id}/companies/{company_id}", for
	// example,
	// "projects/api-test-project/companies/foo".
	//
	// At most 20 company filters are allowed.
	CompanyNames []string `json:"companyNames,omitempty"`

	// CompensationFilter: Optional.
	//
	// This search filter is applied only to
	// Job.compensation_info. For example, if the filter is specified
	// as "Hourly job with per-hour compensation > $15", only jobs
	// meeting
	// these criteria are searched. If a filter isn't defined, all open
	// jobs
	// are searched.
	CompensationFilter *CompensationFilter `json:"compensationFilter,omitempty"`

	// CustomAttributeFilter: Optional.
	//
	// This filter specifies a structured syntax to match against
	// the
	// Job.custom_attributes marked as `filterable`.
	//
	// The syntax for this expression is a subset of SQL syntax.
	//
	// Supported operators are: `=`, `!=`, `<`, `<=`, `>`, and `>=` where
	// the
	// left of the operator is a custom field key and the right of the
	// operator
	// is a number or a quoted string. You must escape backslash (\\)
	// and
	// quote (\") characters.
	//
	// Supported functions are `LOWER([field_name])` to
	// perform a case insensitive match and `EMPTY([field_name])` to filter
	// on the
	// existence of a key.
	//
	// Boolean expressions (AND/OR/NOT) are supported up to 3 levels
	// of
	// nesting (for example, "((A AND B AND C) OR NOT D) AND E"), a maximum
	// of 50
	// comparisons or functions are allowed in the expression. The
	// expression
	// must be < 3000 characters in length.
	//
	// Sample Query:
	// `(LOWER(driving_license)="class \"a\"" OR EMPTY(driving_license))
	// AND
	// driving_years > 10`
	CustomAttributeFilter string `json:"customAttributeFilter,omitempty"`

	// DisableSpellCheck: Optional.
	//
	// This flag controls the spell-check feature. If false, the
	// service attempts to correct a misspelled query,
	// for example, "enginee" is corrected to "engineer".
	//
	// Defaults to false: a spell check is performed.
	DisableSpellCheck bool `json:"disableSpellCheck,omitempty"`

	// EmploymentTypes: Optional.
	//
	// The employment type filter specifies the employment type of jobs
	// to
	// search against, such as EmploymentType.FULL_TIME.
	//
	// If a value is not specified, jobs in the search results includes
	// any
	// employment type.
	//
	// If multiple values are specified, jobs in the search results
	// include
	// any of the specified employment types.
	//
	// Possible values:
	//   "EMPLOYMENT_TYPE_UNSPECIFIED" - The default value if the employment
	// type is not specified.
	//   "FULL_TIME" - The job requires working a number of hours that
	// constitute full
	// time employment, typically 40 or more hours per week.
	//   "PART_TIME" - The job entails working fewer hours than a full time
	// job,
	// typically less than 40 hours a week.
	//   "CONTRACTOR" - The job is offered as a contracted, as opposed to a
	// salaried employee,
	// position.
	//   "CONTRACT_TO_HIRE" - The job is offered as a contracted position
	// with the understanding
	// that it's converted into a full-time position at the end of
	// the
	// contract. Jobs of this type are also returned by a search
	// for
	// EmploymentType.CONTRACTOR jobs.
	//   "TEMPORARY" - The job is offered as a temporary employment
	// opportunity, usually
	// a short-term engagement.
	//   "INTERN" - The job is a fixed-term opportunity for students or
	// entry-level job
	// seekers to obtain on-the-job training, typically offered as a
	// summer
	// position.
	//   "VOLUNTEER" - The is an opportunity for an individual to volunteer,
	// where there's no
	// expectation of compensation for the provided services.
	//   "PER_DIEM" - The job requires an employee to work on an as-needed
	// basis with a
	// flexible schedule.
	//   "FLY_IN_FLY_OUT" - The job involves employing people in remote
	// areas and flying them
	// temporarily to the work site instead of relocating employees and
	// their
	// families permanently.
	//   "OTHER_EMPLOYMENT_TYPE" - The job does not fit any of the other
	// listed types.
	EmploymentTypes []string `json:"employmentTypes,omitempty"`

	// JobCategories: Optional.
	//
	// The category filter specifies the categories of jobs to search
	// against.
	// See Category for more information.
	//
	// If a value is not specified, jobs from any category are searched
	// against.
	//
	// If multiple values are specified, jobs from any of the
	// specified
	// categories are searched against.
	//
	// Possible values:
	//   "JOB_CATEGORY_UNSPECIFIED" - The default value if the category
	// isn't specified.
	//   "ACCOUNTING_AND_FINANCE" - An accounting and finance job, such as
	// an Accountant.
	//   "ADMINISTRATIVE_AND_OFFICE" - An administrative and office job,
	// such as an Administrative Assistant.
	//   "ADVERTISING_AND_MARKETING" - An advertising and marketing job,
	// such as Marketing Manager.
	//   "ANIMAL_CARE" - An animal care job, such as Veterinarian.
	//   "ART_FASHION_AND_DESIGN" - An art, fashion, or design job, such as
	// Designer.
	//   "BUSINESS_OPERATIONS" - A business operations job, such as Business
	// Operations Manager.
	//   "CLEANING_AND_FACILITIES" - A cleaning and facilities job, such as
	// Custodial Staff.
	//   "COMPUTER_AND_IT" - A computer and IT job, such as Systems
	// Administrator.
	//   "CONSTRUCTION" - A construction job, such as General Laborer.
	//   "CUSTOMER_SERVICE" - A customer service job, such s Cashier.
	//   "EDUCATION" - An education job, such as School Teacher.
	//   "ENTERTAINMENT_AND_TRAVEL" - An entertainment and travel job, such
	// as Flight Attendant.
	//   "FARMING_AND_OUTDOORS" - A farming or outdoor job, such as Park
	// Ranger.
	//   "HEALTHCARE" - A healthcare job, such as Registered Nurse.
	//   "HUMAN_RESOURCES" - A human resources job, such as Human Resources
	// Director.
	//   "INSTALLATION_MAINTENANCE_AND_REPAIR" - An installation,
	// maintenance, or repair job, such as Electrician.
	//   "LEGAL" - A legal job, such as Law Clerk.
	//   "MANAGEMENT" - A management job, often used in conjunction with
	// another category,
	// such as Store Manager.
	//   "MANUFACTURING_AND_WAREHOUSE" - A manufacturing or warehouse job,
	// such as Assembly Technician.
	//   "MEDIA_COMMUNICATIONS_AND_WRITING" - A media, communications, or
	// writing job, such as Media Relations.
	//   "OIL_GAS_AND_MINING" - An oil, gas or mining job, such as Offshore
	// Driller.
	//   "PERSONAL_CARE_AND_SERVICES" - A personal care and services job,
	// such as Hair Stylist.
	//   "PROTECTIVE_SERVICES" - A protective services job, such as Security
	// Guard.
	//   "REAL_ESTATE" - A real estate job, such as Buyer's Agent.
	//   "RESTAURANT_AND_HOSPITALITY" - A restaurant and hospitality job,
	// such as Restaurant Server.
	//   "SALES_AND_RETAIL" - A sales and/or retail job, such Sales
	// Associate.
	//   "SCIENCE_AND_ENGINEERING" - A science and engineering job, such as
	// Lab Technician.
	//   "SOCIAL_SERVICES_AND_NON_PROFIT" - A social services or non-profit
	// job, such as Case Worker.
	//   "SPORTS_FITNESS_AND_RECREATION" - A sports, fitness, or recreation
	// job, such as Personal Trainer.
	//   "TRANSPORTATION_AND_LOGISTICS" - A transportation or logistics job,
	// such as Truck Driver.
	JobCategories []string `json:"jobCategories,omitempty"`

	// LanguageCodes: Optional.
	//
	// This filter specifies the locale of jobs to search against,
	// for example, "en-US".
	//
	// If a value isn't specified, the search results can contain jobs in
	// any
	// locale.
	//
	//
	// Language codes should be in BCP-47 format, such as "en-US" or
	// "sr-Latn".
	// For more information, see
	// [Tags for Identifying
	// Languages](https://tools.ietf.org/html/bcp47).
	//
	// At most 10 language code filters are allowed.
	LanguageCodes []string `json:"languageCodes,omitempty"`

	// LocationFilters: Optional.
	//
	// The location filter specifies geo-regions containing the jobs
	// to
	// search against. See LocationFilter for more information.
	//
	// If a location value isn't specified, jobs fitting the other
	// search
	// criteria are retrieved regardless of where they're located.
	//
	// If multiple values are specified, jobs are retrieved from any of
	// the
	// specified locations, and, if different values are specified
	// for the LocationFilter.distance_in_miles parameter, the
	// maximum
	// provided distance is used for all locations.
	//
	// At most 5 location filters are allowed.
	LocationFilters []*LocationFilter `json:"locationFilters,omitempty"`

	// PublishTimeRange: Optional.
	//
	// Jobs published within a range specified by this filter are
	// searched
	// against.
	PublishTimeRange *TimestampRange `json:"publishTimeRange,omitempty"`

	// Query: Optional.
	//
	// The query string that matches against the job title, description,
	// and
	// location fields.
	//
	// The maximum number of allowed characters is 255.
	Query string `json:"query,omitempty"`

	// ForceSendFields is a list of field names (e.g. "CommuteFilter") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "CommuteFilter") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *JobQuery) MarshalJSON() ([]byte, error) {
	type NoMethod JobQuery
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// LatLng: An object representing a latitude/longitude pair. This is
// expressed as a pair
// of doubles representing degrees latitude and degrees longitude.
// Unless
// specified otherwise, this must conform to the
// <a
// href="http://www.unoosa.org/pdf/icg/2012/template/WGS_84.pdf">WGS84
// st
// andard</a>. Values must be within normalized ranges.
type LatLng struct {
	// Latitude: The latitude in degrees. It must be in the range [-90.0,
	// +90.0].
	Latitude float64 `json:"latitude,omitempty"`

	// Longitude: The longitude in degrees. It must be in the range [-180.0,
	// +180.0].
	Longitude float64 `json:"longitude,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Latitude") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Latitude") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *LatLng) MarshalJSON() ([]byte, error) {
	type NoMethod LatLng
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

func (s *LatLng) UnmarshalJSON(data []byte) error {
	type NoMethod LatLng
	var s1 struct {
		Latitude  gensupport.JSONFloat64 `json:"latitude"`
		Longitude gensupport.JSONFloat64 `json:"longitude"`
		*NoMethod
	}
	s1.NoMethod = (*NoMethod)(s)
	if err := json.Unmarshal(data, &s1); err != nil {
		return err
	}
	s.Latitude = float64(s1.Latitude)
	s.Longitude = float64(s1.Longitude)
	return nil
}

// ListCompaniesResponse: Output only.
//
// The List companies response object.
type ListCompaniesResponse struct {
	// Companies: Companies for the current client.
	Companies []*Company `json:"companies,omitempty"`

	// Metadata: Additional information for the API invocation, such as the
	// request
	// tracking id.
	Metadata *ResponseMetadata `json:"metadata,omitempty"`

	// NextPageToken: A token to retrieve the next page of results.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "Companies") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Companies") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *ListCompaniesResponse) MarshalJSON() ([]byte, error) {
	type NoMethod ListCompaniesResponse
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// ListJobsResponse: Output only.
//
// List jobs response.
type ListJobsResponse struct {
	// Jobs: The Jobs for a given company.
	//
	// The maximum number of items returned is based on the limit
	// field
	// provided in the request.
	Jobs []*Job `json:"jobs,omitempty"`

	// Metadata: Additional information for the API invocation, such as the
	// request
	// tracking id.
	Metadata *ResponseMetadata `json:"metadata,omitempty"`

	// NextPageToken: A token to retrieve the next page of results.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "Jobs") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Jobs") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *ListJobsResponse) MarshalJSON() ([]byte, error) {
	type NoMethod ListJobsResponse
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// Location: Output only.
//
// A resource that represents a location with full geographic
// information.
type Location struct {
	// LatLng: An object representing a latitude/longitude pair.
	LatLng *LatLng `json:"latLng,omitempty"`

	// LocationType: The type of a location, which corresponds to the
	// address lines field of
	// PostalAddress. For example, "Downtown, Atlanta, GA, USA" has a type
	// of
	// LocationType#NEIGHBORHOOD, and "Kansas City, KS, USA" has a type
	// of
	// LocationType#LOCALITY.
	//
	// Possible values:
	//   "LOCATION_TYPE_UNSPECIFIED" - Default value if the type is not
	// specified.
	//   "COUNTRY" - A country level location.
	//   "ADMINISTRATIVE_AREA" - A state or equivalent level location.
	//   "SUB_ADMINISTRATIVE_AREA" - A county or equivalent level location.
	//   "LOCALITY" - A city or equivalent level location.
	//   "POSTAL_CODE" - A postal code level location.
	//   "SUB_LOCALITY" - A sublocality is a subdivision of a locality, for
	// example a city borough,
	// ward, or arrondissement. Sublocalities are usually recognized by a
	// local
	// political authority. For example, Manhattan and Brooklyn are
	// recognized
	// as boroughs by the City of New York, and are therefore modeled
	// as
	// sublocalities.
	//   "SUB_LOCALITY_1" - A district or equivalent level location.
	//   "SUB_LOCALITY_2" - A smaller district or equivalent level display.
	//   "NEIGHBORHOOD" - A neighborhood level location.
	//   "STREET_ADDRESS" - A street address level location.
	LocationType string `json:"locationType,omitempty"`

	// PostalAddress: Postal address of the location that includes human
	// readable information,
	// such as postal delivery and payments addresses. Given a postal
	// address,
	// a postal service can deliver items to a premises, P.O. Box, or
	// other
	// delivery location.
	PostalAddress *PostalAddress `json:"postalAddress,omitempty"`

	// RadiusInMiles: Radius in meters of the job location. This value is
	// derived from the
	// location bounding box in which a circle with the specified
	// radius
	// centered from LatLng coves the area associated with the job
	// location.
	// For example, currently, "Mountain View, CA, USA" has a radius of
	// 6.17 miles.
	RadiusInMiles float64 `json:"radiusInMiles,omitempty"`

	// ForceSendFields is a list of field names (e.g. "LatLng") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "LatLng") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *Location) MarshalJSON() ([]byte, error) {
	type NoMethod Location
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

func (s *Location) UnmarshalJSON(data []byte) error {
	type NoMethod Location
	var s1 struct {
		RadiusInMiles gensupport.JSONFloat64 `json:"radiusInMiles"`
		*NoMethod
	}
	s1.NoMethod = (*NoMethod)(s)
	if err := json.Unmarshal(data, &s1); err != nil {
		return err
	}
	s.RadiusInMiles = float64(s1.RadiusInMiles)
	return nil
}

// LocationFilter: Input only.
//
// Geographic region of the search.
type LocationFilter struct {
	// Address: Optional.
	//
	// The address name, such as "Mountain View" or "Bay Area".
	Address string `json:"address,omitempty"`

	// DistanceInMiles: Optional.
	//
	//
	// The distance_in_miles is applied when the location being searched for
	// is
	// identified as a city or smaller. When the location being searched for
	// is a
	// state or larger, this field is ignored.
	DistanceInMiles float64 `json:"distanceInMiles,omitempty"`

	// LatLng: Optional.
	//
	// The latitude and longitude of the geographic center from which
	// to
	// search. This field's ignored if `address` is provided.
	LatLng *LatLng `json:"latLng,omitempty"`

	// RegionCode: Optional.
	//
	// CLDR region code of the country/region of the address. This is
	// used
	// to address ambiguity of the user-input location, for example,
	// "Liverpool"
	// against "Liverpool, NY, US" or "Liverpool, UK".
	//
	// Set this field if all the jobs to search against are from a same
	// region,
	// or jobs are world-wide, but the job seeker is from a specific
	// region.
	//
	// See http://cldr.unicode.org/
	// and
	// http://www.unicode.org/cldr/charts/30/supplemental/territory_infor
	// mation.html
	// for details. Example: "CH" for Switzerland.
	RegionCode string `json:"regionCode,omitempty"`

	// TelecommutePreference: Optional.
	//
	// Allows the client to return jobs without a
	// set location, specifically, telecommuting jobs (telecomuting is
	// considered
	// by the service as a special location.
	// Job.posting_region indicates if a job permits telecommuting.
	// If this field is set to
	// TelecommutePreference.TELECOMMUTE_ALLOWED,
	// telecommuting jobs are searched, and address and lat_lng are
	// ignored. If not set or set
	// to
	// TelecommutePreference.TELECOMMUTE_EXCLUDED, telecommute job are
	// not
	// searched.
	//
	// This filter can be used by itself to search exclusively for
	// telecommuting
	// jobs, or it can be combined with another location
	// filter to search for a combination of job locations,
	// such as "Mountain View" or "telecommuting" jobs. However, when used
	// in
	// combination with other location filters, telecommuting jobs can
	// be
	// treated as less relevant than other jobs in the search response.
	//
	// Possible values:
	//   "TELECOMMUTE_PREFERENCE_UNSPECIFIED" - Default value if the
	// telecommute preference is not specified.
	//   "TELECOMMUTE_EXCLUDED" - Exclude telecommute jobs.
	//   "TELECOMMUTE_ALLOWED" - Allow telecommute jobs.
	TelecommutePreference string `json:"telecommutePreference,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Address") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Address") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *LocationFilter) MarshalJSON() ([]byte, error) {
	type NoMethod LocationFilter
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

func (s *LocationFilter) UnmarshalJSON(data []byte) error {
	type NoMethod LocationFilter
	var s1 struct {
		DistanceInMiles gensupport.JSONFloat64 `json:"distanceInMiles"`
		*NoMethod
	}
	s1.NoMethod = (*NoMethod)(s)
	if err := json.Unmarshal(data, &s1); err != nil {
		return err
	}
	s.DistanceInMiles = float64(s1.DistanceInMiles)
	return nil
}

// MatchingJob: Output only.
//
// Job entry with metadata inside SearchJobsResponse.
type MatchingJob struct {
	// CommuteInfo: Commute information which is generated based on
	// specified
	//  CommuteFilter.
	CommuteInfo *CommuteInfo `json:"commuteInfo,omitempty"`

	// Job: Job resource that matches the specified SearchJobsRequest.
	Job *Job `json:"job,omitempty"`

	// JobSummary: A summary of the job with core information that's
	// displayed on the search
	// results listing page.
	JobSummary string `json:"jobSummary,omitempty"`

	// JobTitleSnippet: Contains snippets of text from the Job.job_title
	// field most
	// closely matching a search query's keywords, if available. The
	// matching
	// query keywords are enclosed in HTML bold tags.
	JobTitleSnippet string `json:"jobTitleSnippet,omitempty"`

	// SearchTextSnippet: Contains snippets of text from the Job.description
	// and similar
	// fields that most closely match a search query's keywords, if
	// available.
	// All HTML tags in the original fields are stripped when returned in
	// this
	// field, and matching query keywords are enclosed in HTML bold tags.
	SearchTextSnippet string `json:"searchTextSnippet,omitempty"`

	// ForceSendFields is a list of field names (e.g. "CommuteInfo") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "CommuteInfo") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *MatchingJob) MarshalJSON() ([]byte, error) {
	type NoMethod MatchingJob
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// Money: Represents an amount of money with its currency type.
type Money struct {
	// CurrencyCode: The 3-letter currency code defined in ISO 4217.
	CurrencyCode string `json:"currencyCode,omitempty"`

	// Nanos: Number of nano (10^-9) units of the amount.
	// The value must be between -999,999,999 and +999,999,999 inclusive.
	// If `units` is positive, `nanos` must be positive or zero.
	// If `units` is zero, `nanos` can be positive, zero, or negative.
	// If `units` is negative, `nanos` must be negative or zero.
	// For example $-1.75 is represented as `units`=-1 and
	// `nanos`=-750,000,000.
	Nanos int64 `json:"nanos,omitempty"`

	// Units: The whole units of the amount.
	// For example if `currencyCode` is "USD", then 1 unit is one US
	// dollar.
	Units int64 `json:"units,omitempty,string"`

	// ForceSendFields is a list of field names (e.g. "CurrencyCode") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "CurrencyCode") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *Money) MarshalJSON() ([]byte, error) {
	type NoMethod Money
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// NumericBucketingOption: Input only.
//
// Use this field to specify bucketing option for the histogram search
// response.
type NumericBucketingOption struct {
	// BucketBounds: Required.
	//
	// Two adjacent values form a histogram bucket. Values should be
	// in
	// ascending order. For example, if [5, 10, 15] are provided, four
	// buckets are
	// created: (-inf, 5), 5, 10), [10, 15), [15, inf). At most
	// 20
	// [buckets_bound is supported.
	BucketBounds []float64 `json:"bucketBounds,omitempty"`

	// RequiresMinMax: Optional.
	//
	// If set to true, the histogram result includes minimum/maximum
	// value of the numeric field.
	RequiresMinMax bool `json:"requiresMinMax,omitempty"`

	// ForceSendFields is a list of field names (e.g. "BucketBounds") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "BucketBounds") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *NumericBucketingOption) MarshalJSON() ([]byte, error) {
	type NoMethod NumericBucketingOption
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// NumericBucketingResult: Output only.
//
// Custom numeric bucketing result.
type NumericBucketingResult struct {
	// Counts: Count within each bucket. Its size is the length
	// of
	// NumericBucketingOption.bucket_bounds plus 1.
	Counts []*BucketizedCount `json:"counts,omitempty"`

	// MaxValue: Stores the maximum value of the numeric field. Is populated
	// only if
	// [NumericBucketingOption.requires_min_max] is set to true.
	MaxValue float64 `json:"maxValue,omitempty"`

	// MinValue: Stores the minimum value of the numeric field. Will be
	// populated only if
	// [NumericBucketingOption.requires_min_max] is set to true.
	MinValue float64 `json:"minValue,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Counts") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Counts") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *NumericBucketingResult) MarshalJSON() ([]byte, error) {
	type NoMethod NumericBucketingResult
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

func (s *NumericBucketingResult) UnmarshalJSON(data []byte) error {
	type NoMethod NumericBucketingResult
	var s1 struct {
		MaxValue gensupport.JSONFloat64 `json:"maxValue"`
		MinValue gensupport.JSONFloat64 `json:"minValue"`
		*NoMethod
	}
	s1.NoMethod = (*NoMethod)(s)
	if err := json.Unmarshal(data, &s1); err != nil {
		return err
	}
	s.MaxValue = float64(s1.MaxValue)
	s.MinValue = float64(s1.MinValue)
	return nil
}

// PostalAddress: Represents a postal address, e.g. for postal delivery
// or payments addresses.
// Given a postal address, a postal service can deliver items to a
// premise, P.O.
// Box or similar.
// It is not intended to model geographical locations (roads,
// towns,
// mountains).
//
// In typical usage an address would be created via user input or from
// importing
// existing data, depending on the type of process.
//
// Advice on address input / editing:
//  - Use an i18n-ready address widget such as
//    https://github.com/googlei18n/libaddressinput)
// - Users should not be presented with UI elements for input or editing
// of
//   fields outside countries where that field is used.
//
// For more guidance on how to use this schema, please
// see:
// https://support.google.com/business/answer/6397478
type PostalAddress struct {
	// AddressLines: Unstructured address lines describing the lower levels
	// of an address.
	//
	// Because values in address_lines do not have type information and
	// may
	// sometimes contain multiple values in a single field (e.g.
	// "Austin, TX"), it is important that the line order is clear. The
	// order of
	// address lines should be "envelope order" for the country/region of
	// the
	// address. In places where this can vary (e.g. Japan), address_language
	// is
	// used to make it explicit (e.g. "ja" for large-to-small ordering
	// and
	// "ja-Latn" or "en" for small-to-large). This way, the most specific
	// line of
	// an address can be selected based on the language.
	//
	// The minimum permitted structural representation of an address
	// consists
	// of a region_code with all remaining information placed in
	// the
	// address_lines. It would be possible to format such an address
	// very
	// approximately without geocoding, but no semantic reasoning could
	// be
	// made about any of the address components until it was at
	// least
	// partially resolved.
	//
	// Creating an address only containing a region_code and address_lines,
	// and
	// then geocoding is the recommended way to handle completely
	// unstructured
	// addresses (as opposed to guessing which parts of the address should
	// be
	// localities or administrative areas).
	AddressLines []string `json:"addressLines,omitempty"`

	// AdministrativeArea: Optional. Highest administrative subdivision
	// which is used for postal
	// addresses of a country or region.
	// For example, this can be a state, a province, an oblast, or a
	// prefecture.
	// Specifically, for Spain this is the province and not the
	// autonomous
	// community (e.g. "Barcelona" and not "Catalonia").
	// Many countries don't use an administrative area in postal addresses.
	// E.g.
	// in Switzerland this should be left unpopulated.
	AdministrativeArea string `json:"administrativeArea,omitempty"`

	// LanguageCode: Optional. BCP-47 language code of the contents of this
	// address (if
	// known). This is often the UI language of the input form or is
	// expected
	// to match one of the languages used in the address' country/region, or
	// their
	// transliterated equivalents.
	// This can affect formatting in certain countries, but is not
	// critical
	// to the correctness of the data and will never affect any validation
	// or
	// other non-formatting related operations.
	//
	// If this value is not known, it should be omitted (rather than
	// specifying a
	// possibly incorrect default).
	//
	// Examples: "zh-Hant", "ja", "ja-Latn", "en".
	LanguageCode string `json:"languageCode,omitempty"`

	// Locality: Optional. Generally refers to the city/town portion of the
	// address.
	// Examples: US city, IT comune, UK post town.
	// In regions of the world where localities are not well defined or do
	// not fit
	// into this structure well, leave locality empty and use address_lines.
	Locality string `json:"locality,omitempty"`

	// Organization: Optional. The name of the organization at the address.
	Organization string `json:"organization,omitempty"`

	// PostalCode: Optional. Postal code of the address. Not all countries
	// use or require
	// postal codes to be present, but where they are used, they may
	// trigger
	// additional validation with other parts of the address (e.g.
	// state/zip
	// validation in the U.S.A.).
	PostalCode string `json:"postalCode,omitempty"`

	// Recipients: Optional. The recipient at the address.
	// This field may, under certain circumstances, contain multiline
	// information.
	// For example, it might contain "care of" information.
	Recipients []string `json:"recipients,omitempty"`

	// RegionCode: Required. CLDR region code of the country/region of the
	// address. This
	// is never inferred and it is up to the user to ensure the value
	// is
	// correct. See http://cldr.unicode.org/
	// and
	// http://www.unicode.org/cldr/charts/30/supplemental/territory_infor
	// mation.html
	// for details. Example: "CH" for Switzerland.
	RegionCode string `json:"regionCode,omitempty"`

	// Revision: The schema revision of the `PostalAddress`. This must be
	// set to 0, which is
	// the latest revision.
	//
	// All new revisions **must** be backward compatible with old revisions.
	Revision int64 `json:"revision,omitempty"`

	// SortingCode: Optional. Additional, country-specific, sorting code.
	// This is not used
	// in most regions. Where it is used, the value is either a string
	// like
	// "CEDEX", optionally followed by a number (e.g. "CEDEX 7"), or just a
	// number
	// alone, representing the "sector code" (Jamaica), "delivery area
	// indicator"
	// (Malawi) or "post office indicator" (e.g. Côte d'Ivoire).
	SortingCode string `json:"sortingCode,omitempty"`

	// Sublocality: Optional. Sublocality of the address.
	// For example, this can be neighborhoods, boroughs, districts.
	Sublocality string `json:"sublocality,omitempty"`

	// ForceSendFields is a list of field names (e.g. "AddressLines") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "AddressLines") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *PostalAddress) MarshalJSON() ([]byte, error) {
	type NoMethod PostalAddress
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// ProcessingOptions: Input only.
//
// Options for job processing.
type ProcessingOptions struct {
	// DisableStreetAddressResolution: Optional.
	//
	// If set to `true`, the service does not attempt to resolve a
	// more precise address for the job.
	DisableStreetAddressResolution bool `json:"disableStreetAddressResolution,omitempty"`

	// HtmlSanitization: Optional.
	//
	// Option for job HTML content sanitization. Applied fields are:
	//
	// * description
	// * applicationInfo.instruction
	// * incentives
	// * qualifications
	// * responsibilities
	//
	// HTML tags in these fields may be stripped if sanitiazation is
	// not
	// disabled.
	//
	// Defaults to HtmlSanitization.SIMPLE_FORMATTING_ONLY.
	//
	// Possible values:
	//   "HTML_SANITIZATION_UNSPECIFIED" - Default value.
	//   "HTML_SANITIZATION_DISABLED" - Disables sanitization on HTML input.
	//   "SIMPLE_FORMATTING_ONLY" - Sanitizes HTML input, only accepts bold,
	// italic, ordered list, and
	// unordered list markup tags.
	HtmlSanitization string `json:"htmlSanitization,omitempty"`

	// ForceSendFields is a list of field names (e.g.
	// "DisableStreetAddressResolution") to unconditionally include in API
	// requests. By default, fields with empty values are omitted from API
	// requests. However, any non-pointer, non-interface field appearing in
	// ForceSendFields will be sent to the server regardless of whether the
	// field is empty or not. This may be used to include empty fields in
	// Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g.
	// "DisableStreetAddressResolution") to include in API requests with the
	// JSON null value. By default, fields with empty values are omitted
	// from API requests. However, any field with an empty value appearing
	// in NullFields will be sent to the server as null. It is an error if a
	// field in this list has a non-empty value. This may be used to include
	// null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *ProcessingOptions) MarshalJSON() ([]byte, error) {
	type NoMethod ProcessingOptions
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// RequestMetadata: Input only.
//
// Meta information related to the job searcher or entity
// conducting the job search. This information is used to improve
// the
// performance of the service.
type RequestMetadata struct {
	// DeviceInfo: Optional.
	//
	// The type of device used by the job seeker at the time of the call to
	// the
	// service.
	DeviceInfo *DeviceInfo `json:"deviceInfo,omitempty"`

	// Domain: Required.
	//
	// The client-defined scope or source of the service call, which
	// typically
	// is the domain on
	// which the service has been implemented and is currently being
	// run.
	//
	// For example, if the service is being run by client <em>Foo,
	// Inc.</em>, on
	// job board www.foo.com and career site www.bar.com, then this field
	// is
	// set to "foo.com" for use on the job board, and "bar.com" for use on
	// the
	// career site.
	//
	// If this field isn't available for some reason, send "UNKNOWN".
	// Any improvements to the model for a particular tenant site rely on
	// this
	// field being set correctly to a domain.
	//
	// The maximum number of allowed characters is 255.
	Domain string `json:"domain,omitempty"`

	// SessionId: Required.
	//
	// A unique session identification string. A session is defined as
	// the
	// duration of an end user's interaction with the service over a
	// certain
	// period.
	// Obfuscate this field for privacy concerns before
	// providing it to the service.
	//
	// If this field is not available for some reason, send "UNKNOWN".
	// Note
	// that any improvements to the model for a particular tenant
	// site, rely on this field being set correctly to some unique
	// session_id.
	//
	// The maximum number of allowed characters is 255.
	SessionId string `json:"sessionId,omitempty"`

	// UserId: Required.
	//
	// A unique user identification string, as determined by the client.
	// To have the strongest positive impact on search quality
	// make sure the client-level is unique.
	// Obfuscate this field for privacy concerns before
	// providing it to the service.
	//
	// If this field is not available for some reason, send "UNKNOWN".
	// Note
	// that any improvements to the model for a particular tenant
	// site, rely on this field being set correctly to a unique
	// user_id.
	//
	// The maximum number of allowed characters is 255.
	UserId string `json:"userId,omitempty"`

	// ForceSendFields is a list of field names (e.g. "DeviceInfo") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "DeviceInfo") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *RequestMetadata) MarshalJSON() ([]byte, error) {
	type NoMethod RequestMetadata
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// ResponseMetadata: Output only.
//
// Additional information returned to client, such as debugging
// information.
type ResponseMetadata struct {
	// RequestId: A unique id associated with this call.
	// This id is logged for tracking purposes.
	RequestId string `json:"requestId,omitempty"`

	// ForceSendFields is a list of field names (e.g. "RequestId") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "RequestId") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *ResponseMetadata) MarshalJSON() ([]byte, error) {
	type NoMethod ResponseMetadata
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// SearchJobsRequest: Input only.
//
// The Request body of the `SearchJobs` call.
type SearchJobsRequest struct {
	// DisableKeywordMatch: Optional.
	//
	// Controls whether to disable exact keyword match on
	// Job.job_title,
	// Job.description, Job.company_display_name,
	// Job.locations,
	// Job.qualifications. When disable keyword match is turned off,
	// a
	// keyword match returns jobs that do not match given category filters
	// when
	// there are matching keywords. For example, the query "program
	// manager," a
	// result is returned even if the job posting has the title
	// "software
	// developer," which does not fall into "program manager" ontology, but
	// does
	// have "program manager" appearing in its description.
	//
	// For queries like "cloud" that does not contain title or
	// location specific ontology, jobs with "cloud" keyword matches are
	// returned
	// regardless of this flag's value.
	//
	// Please use Company.keyword_searchable_custom_fields
	// or
	// Company.keyword_searchable_custom_attributes if company
	// specific
	// globally matched custom field/attribute string values is needed.
	// Enabling
	// keyword match improves recall of subsequent search
	// requests.
	//
	// Defaults to false.
	DisableKeywordMatch bool `json:"disableKeywordMatch,omitempty"`

	// EnableBroadening: Optional.
	//
	// Controls whether to broaden the search when it produces sparse
	// results.
	// Broadened queries append results to the end of the matching
	// results
	// list.
	//
	// Defaults to false.
	EnableBroadening bool `json:"enableBroadening,omitempty"`

	// HistogramFacets: Optional.
	//
	// Histogram requests for jobs matching JobQuery.
	HistogramFacets *HistogramFacets `json:"histogramFacets,omitempty"`

	// JobQuery: Optional.
	//
	// Query used to search against jobs, such as keyword, location filters,
	// etc.
	JobQuery *JobQuery `json:"jobQuery,omitempty"`

	// JobView: Optional.
	//
	// The desired job attributes returned for jobs in the
	// search response. Defaults to JobView.SMALL if no value is specified.
	//
	// Possible values:
	//   "JOB_VIEW_UNSPECIFIED" - Default value.
	//   "JOB_VIEW_ID_ONLY" - A ID only view of job, with following
	// attributes:
	// Job.name, Job.requisition_id, Job.language_code.
	//   "JOB_VIEW_MINIMAL" - A minimal view of the job, with the following
	// attributes:
	// Job.name, Job.requisition_id, Job.job_title,
	// Job.company_name, Job.DerivedInfo.locations, Job.language_code.
	//   "JOB_VIEW_SMALL" - A small view of the job, with the following
	// attributes in the search
	// results: Job.name, Job.requisition_id,
	// Job.job_title,
	// Job.company_name, Job.DerivedInfo.locations,
	// Job.visibility,
	// Job.language_code, Job.description.
	//   "JOB_VIEW_FULL" - All available attributes are included in the
	// search results.
	JobView string `json:"jobView,omitempty"`

	// Offset: Optional.
	//
	// An integer that specifies the current offset (that is, starting
	// result
	// location, amongst the jobs deemed by the API as relevant) in
	// search
	// results. This field is only considered if page_token is unset.
	//
	// For example, 0 means to  return results starting from the first
	// matching
	// job, and 10 means to return from the 11th job. This can be used
	// for
	// pagination, (for example, pageSize = 10 and offset = 10 means to
	// return
	// from the second page).
	Offset int64 `json:"offset,omitempty"`

	// OrderBy: Optional.
	//
	// The criteria determining how search results are sorted. Default
	// is
	// "relevance desc".
	//
	// Supported options are:
	//
	// * "relevance desc": By relevance descending, as determined by the
	// API
	// algorithms. Relevance thresholding of query results is only
	// available
	// with this ordering.
	// * "posting`_`publish`_`time desc": By Job.posting_publish_time
	// descending.
	// * "posting`_`update`_`time desc": By Job.posting_update_time
	// descending.
	// * "title": By Job.title ascending.
	// * "title desc": By Job.title descending.
	// * "annualized`_`base`_`compensation": By
	// job's
	// CompensationInfo.annualized_base_compensation_range ascending.
	// Jobs
	// whose annualized base compensation is unspecified are put at the end
	// of
	// search results.
	// * "annualized`_`base`_`compensation desc": By
	// job's
	// CompensationInfo.annualized_base_compensation_range descending.
	// Jobs
	// whose annualized base compensation is unspecified are put at the end
	// of
	// search results.
	// * "annualized`_`total`_`compensation": By
	// job's
	// CompensationInfo.annualized_total_compensation_range ascending.
	// Jobs
	// whose annualized base compensation is unspecified are put at the end
	// of
	// search results.
	// * "annualized`_`total`_`compensation desc": By
	// job's
	// CompensationInfo.annualized_total_compensation_range descending.
	// Jobs
	// whose annualized base compensation is unspecified are put at the end
	// of
	// search results.
	OrderBy string `json:"orderBy,omitempty"`

	// PageSize: Optional.
	//
	// A limit on the number of jobs returned in the search
	// results.
	// Increasing this value above the default value of 10 can increase
	// search
	// response time. The value can be between 1 and 100.
	PageSize int64 `json:"pageSize,omitempty"`

	// PageToken: Optional.
	//
	// The token specifying the current offset within
	// search results. See SearchJobsResponse.next_page_token for
	// an explanation of how to obtain the next set of query results.
	PageToken string `json:"pageToken,omitempty"`

	// RequestMetadata: Required.
	//
	// The meta information collected about the job searcher, used to
	// improve the
	// search quality of the service.. The identifiers, (such as `user_id`)
	// are
	// provided by users, and must be unique and consistent.
	RequestMetadata *RequestMetadata `json:"requestMetadata,omitempty"`

	// RequirePreciseResultSize: Optional.
	//
	// Controls if the search job request requires the return of a
	// precise
	// count of the first 300 results. Setting this to `true`
	// ensures
	// consistency in the number of results per page. Best practice is to
	// set this
	// value to true if a client allows users to jump directly to
	// a
	// non-sequential search results page.
	//
	// Enabling this flag may adversely impact performance.
	//
	// Defaults to false.
	RequirePreciseResultSize bool `json:"requirePreciseResultSize,omitempty"`

	// SearchMode: Optional.
	//
	// Mode of a search.
	//
	// Defaults to SearchMode.JOB_SEARCH.
	//
	// Possible values:
	//   "SEARCH_MODE_UNSPECIFIED" - The mode of the search method isn't
	// specified.
	//   "JOB_SEARCH" - The job search matches against all jobs, and
	// featured jobs
	// (jobs with promotionValue > 0) are not specially handled.
	//   "FEATURED_JOB_SEARCH" - The job search matches only against
	// featured jobs (jobs with a
	// promotionValue > 0). This method doesn't return any jobs having
	// a
	// promotionValue <= 0. The search results order is determined by
	// the
	// promotionValue (jobs with a higher promotionValue are returned higher
	// up
	// in the search results), with relevance being used as a tiebreaker.
	SearchMode string `json:"searchMode,omitempty"`

	// ForceSendFields is a list of field names (e.g. "DisableKeywordMatch")
	// to unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "DisableKeywordMatch") to
	// include in API requests with the JSON null value. By default, fields
	// with empty values are omitted from API requests. However, any field
	// with an empty value appearing in NullFields will be sent to the
	// server as null. It is an error if a field in this list has a
	// non-empty value. This may be used to include null fields in Patch
	// requests.
	NullFields []string `json:"-"`
}

func (s *SearchJobsRequest) MarshalJSON() ([]byte, error) {
	type NoMethod SearchJobsRequest
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// SearchJobsResponse: Output only.
//
// Response for SearchJob method.
type SearchJobsResponse struct {
	// BroadenedQueryJobsCount: If query broadening is enabled, we may
	// append additional results from the
	// broadened query. This number indicates how many of the jobs returned
	// in the
	// jobs field are from the broadened query. These results are always at
	// the
	// end of the jobs list. In particular, a value of 0, or if the field
	// isn't
	// set, all the jobs in the jobs list are from the original
	// (without broadening) query. If this field is non-zero, subsequent
	// requests
	// with offset after this result set should contain all broadened
	// results.
	BroadenedQueryJobsCount int64 `json:"broadenedQueryJobsCount,omitempty"`

	// EstimatedTotalSize: An estimation of the number of jobs that match
	// the specified query.
	//
	// This number is not guaranteed to be accurate. For accurate
	// results,
	// see enable_precise_result_size.
	EstimatedTotalSize int64 `json:"estimatedTotalSize,omitempty"`

	// HistogramResults: The histogram results that match
	// specified
	// SearchJobsRequest.histogram_facets.
	HistogramResults *HistogramResults `json:"histogramResults,omitempty"`

	// LocationFilters: The location filters that the service applied to the
	// specified query. If
	// any filters are lat-lng based, the JobLocation.location_type
	// is
	// JobLocation.LocationType#LOCATION_TYPE_UNSPECIFIED.
	LocationFilters []*Location `json:"locationFilters,omitempty"`

	// MatchingJobs: The Job entities that match the specified
	// SearchJobsRequest.
	MatchingJobs []*MatchingJob `json:"matchingJobs,omitempty"`

	// Metadata: Additional information for the API invocation, such as the
	// request
	// tracking id.
	Metadata *ResponseMetadata `json:"metadata,omitempty"`

	// NextPageToken: The token that specifies the starting position of the
	// next page of results.
	// This field is empty if there are no more results.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// SpellCorrection: The spell checking result, and correction.
	SpellCorrection *SpellingCorrection `json:"spellCorrection,omitempty"`

	// TotalSize: The precise result count, which is available only if the
	// client set
	// enable_precise_result_size to `true` or if the response
	// is the last page of results. Otherwise, the value is `-1`.
	TotalSize int64 `json:"totalSize,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g.
	// "BroadenedQueryJobsCount") to unconditionally include in API
	// requests. By default, fields with empty values are omitted from API
	// requests. However, any non-pointer, non-interface field appearing in
	// ForceSendFields will be sent to the server regardless of whether the
	// field is empty or not. This may be used to include empty fields in
	// Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "BroadenedQueryJobsCount")
	// to include in API requests with the JSON null value. By default,
	// fields with empty values are omitted from API requests. However, any
	// field with an empty value appearing in NullFields will be sent to the
	// server as null. It is an error if a field in this list has a
	// non-empty value. This may be used to include null fields in Patch
	// requests.
	NullFields []string `json:"-"`
}

func (s *SearchJobsResponse) MarshalJSON() ([]byte, error) {
	type NoMethod SearchJobsResponse
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// SpellingCorrection: Output only.
//
// Spell check result.
type SpellingCorrection struct {
	// Corrected: Indicates if the query was corrected by the spell checker.
	Corrected bool `json:"corrected,omitempty"`

	// CorrectedText: Correction output consisting of the corrected keyword
	// string.
	CorrectedText string `json:"correctedText,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Corrected") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Corrected") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *SpellingCorrection) MarshalJSON() ([]byte, error) {
	type NoMethod SpellingCorrection
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// TimeOfDay: Represents a time of day. The date and time zone are
// either not significant
// or are specified elsewhere. An API may choose to allow leap seconds.
// Related
// types are google.type.Date and `google.protobuf.Timestamp`.
type TimeOfDay struct {
	// Hours: Hours of day in 24 hour format. Should be from 0 to 23. An API
	// may choose
	// to allow the value "24:00:00" for scenarios like business closing
	// time.
	Hours int64 `json:"hours,omitempty"`

	// Minutes: Minutes of hour of day. Must be from 0 to 59.
	Minutes int64 `json:"minutes,omitempty"`

	// Nanos: Fractions of seconds in nanoseconds. Must be from 0 to
	// 999,999,999.
	Nanos int64 `json:"nanos,omitempty"`

	// Seconds: Seconds of minutes of the time. Must normally be from 0 to
	// 59. An API may
	// allow the value 60 if it allows leap-seconds.
	Seconds int64 `json:"seconds,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Hours") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Hours") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *TimeOfDay) MarshalJSON() ([]byte, error) {
	type NoMethod TimeOfDay
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// TimestampRange: Message representing a period of time between two
// timestamps.
type TimestampRange struct {
	// EndTime: End of the period.
	EndTime string `json:"endTime,omitempty"`

	// StartTime: Begin of the period.
	StartTime string `json:"startTime,omitempty"`

	// ForceSendFields is a list of field names (e.g. "EndTime") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "EndTime") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *TimestampRange) MarshalJSON() ([]byte, error) {
	type NoMethod TimestampRange
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// UpdateCompanyRequest: Input only.
//
// Request for updating a specified company.
type UpdateCompanyRequest struct {
	// Company: Required.
	//
	// The company resource to replace the current resource in the system.
	Company *Company `json:"company,omitempty"`

	// UpdateMask: Optional but strongly recommended for the best
	// service
	// experience.
	//
	// If update_mask is provided, only the specified fields in
	// company are updated. Otherwise all the fields are updated.
	//
	// A field mask to specify the company fields to be updated. Only
	// top level fields of Company are supported.
	UpdateMask string `json:"updateMask,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Company") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Company") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *UpdateCompanyRequest) MarshalJSON() ([]byte, error) {
	type NoMethod UpdateCompanyRequest
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// UpdateJobRequest: Input only.
//
// Update job request.
type UpdateJobRequest struct {
	// Job: Required.
	//
	// The Job to be updated.
	Job *Job `json:"job,omitempty"`

	// UpdateMask: Optional but strongly recommended to be provided for the
	// best service
	// experience.
	//
	// If update_mask is provided, only the specified fields in
	// job are updated. Otherwise all the fields are updated.
	//
	// A field mask to restrict the fields that are updated. Only
	// top level fields of Job are supported.
	UpdateMask string `json:"updateMask,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Job") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Job") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *UpdateJobRequest) MarshalJSON() ([]byte, error) {
	type NoMethod UpdateJobRequest
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// method id "jobs.projects.complete":

type ProjectsCompleteCall struct {
	s            *Service
	name         string
	urlParams_   gensupport.URLParams
	ifNoneMatch_ string
	ctx_         context.Context
	header_      http.Header
}

// Complete: Completes the specified prefix with keyword
// suggestions.
// Intended for use by a job search auto-complete search box.
func (r *ProjectsService) Complete(name string) *ProjectsCompleteCall {
	c := &ProjectsCompleteCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.name = name
	return c
}

// CompanyName sets the optional parameter "companyName": If provided,
// restricts completion to specified company.
//
// The format is "projects/{project_id}/companies/{company_id}", for
// example,
// "projects/api-test-project/companies/foo".
func (c *ProjectsCompleteCall) CompanyName(companyName string) *ProjectsCompleteCall {
	c.urlParams_.Set("companyName", companyName)
	return c
}

// LanguageCode sets the optional parameter "languageCode":
// Required.
//
// The language of the query. This is
// the BCP-47 language code, such as "en-US" or "sr-Latn".
// For more information, see
// [Tags for Identifying
// Languages](https://tools.ietf.org/html/bcp47).
//
// For CompletionType.JOB_TITLE type, only open jobs with
// same
// language_code are returned.
//
// For CompletionType.COMPANY_NAME type,
// only companies having open jobs with same language_code
// are
// returned.
//
// For CompletionType.COMBINED type, only open jobs with
// same
// language_code or companies having open jobs with same
// language_code are returned.
//
// The maximum number of allowed characters is 255.
func (c *ProjectsCompleteCall) LanguageCode(languageCode string) *ProjectsCompleteCall {
	c.urlParams_.Set("languageCode", languageCode)
	return c
}

// PageSize sets the optional parameter "pageSize":
// Required.
//
// Completion result count.
//
// The maximum allowed page size is 10.
func (c *ProjectsCompleteCall) PageSize(pageSize int64) *ProjectsCompleteCall {
	c.urlParams_.Set("pageSize", fmt.Sprint(pageSize))
	return c
}

// Query sets the optional parameter "query": Required.
//
// The query used to generate suggestions.
//
// The maximum number of allowed characters is 255.
func (c *ProjectsCompleteCall) Query(query string) *ProjectsCompleteCall {
	c.urlParams_.Set("query", query)
	return c
}

// Scope sets the optional parameter "scope": The scope of the
// completion. The defaults is CompletionScope.PUBLIC.
//
// Possible values:
//   "COMPLETION_SCOPE_UNSPECIFIED"
//   "TENANT"
//   "PUBLIC"
func (c *ProjectsCompleteCall) Scope(scope string) *ProjectsCompleteCall {
	c.urlParams_.Set("scope", scope)
	return c
}

// Type sets the optional parameter "type": The completion topic. The
// default is CompletionType.COMBINED.
//
// Possible values:
//   "COMPLETION_TYPE_UNSPECIFIED"
//   "JOB_TITLE"
//   "COMPANY_NAME"
//   "COMBINED"
func (c *ProjectsCompleteCall) Type(type_ string) *ProjectsCompleteCall {
	c.urlParams_.Set("type", type_)
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsCompleteCall) Fields(s ...googleapi.Field) *ProjectsCompleteCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// IfNoneMatch sets the optional parameter which makes the operation
// fail if the object's ETag matches the given value. This is useful for
// getting updates only after the object has changed since the last
// request. Use googleapi.IsNotModified to check whether the response
// error from Do is the result of In-None-Match.
func (c *ProjectsCompleteCall) IfNoneMatch(entityTag string) *ProjectsCompleteCall {
	c.ifNoneMatch_ = entityTag
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsCompleteCall) Context(ctx context.Context) *ProjectsCompleteCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsCompleteCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsCompleteCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	if c.ifNoneMatch_ != "" {
		reqHeaders.Set("If-None-Match", c.ifNoneMatch_)
	}
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+name}:complete")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"name": c.name,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.complete" call.
// Exactly one of *CompleteQueryResponse or error will be non-nil. Any
// non-2xx status code is an error. Response headers are in either
// *CompleteQueryResponse.ServerResponse.Header or (if a response was
// returned at all) in error.(*googleapi.Error).Header. Use
// googleapi.IsNotModified to check whether the returned error was
// because http.StatusNotModified was returned.
func (c *ProjectsCompleteCall) Do(opts ...googleapi.CallOption) (*CompleteQueryResponse, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &CompleteQueryResponse{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Completes the specified prefix with keyword suggestions.\nIntended for use by a job search auto-complete search box.",
	//   "flatPath": "v3/projects/{projectsId}:complete",
	//   "httpMethod": "GET",
	//   "id": "jobs.projects.complete",
	//   "parameterOrder": [
	//     "name"
	//   ],
	//   "parameters": {
	//     "companyName": {
	//       "description": "Optional.\n\nIf provided, restricts completion to specified company.\n\nThe format is \"projects/{project_id}/companies/{company_id}\", for example,\n\"projects/api-test-project/companies/foo\".",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "languageCode": {
	//       "description": "Required.\n\nThe language of the query. This is\nthe BCP-47 language code, such as \"en-US\" or \"sr-Latn\".\nFor more information, see\n[Tags for Identifying Languages](https://tools.ietf.org/html/bcp47).\n\nFor CompletionType.JOB_TITLE type, only open jobs with same\nlanguage_code are returned.\n\nFor CompletionType.COMPANY_NAME type,\nonly companies having open jobs with same language_code are\nreturned.\n\nFor CompletionType.COMBINED type, only open jobs with same\nlanguage_code or companies having open jobs with same\nlanguage_code are returned.\n\nThe maximum number of allowed characters is 255.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "name": {
	//       "description": "Required.\n\nResource name of project the completion is performed within.\n\nThe format is \"projects/{project_id}\", for example,\n\"projects/api-test-project\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "pageSize": {
	//       "description": "Required.\n\nCompletion result count.\n\nThe maximum allowed page size is 10.",
	//       "format": "int32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "query": {
	//       "description": "Required.\n\nThe query used to generate suggestions.\n\nThe maximum number of allowed characters is 255.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "scope": {
	//       "description": "Optional.\n\nThe scope of the completion. The defaults is CompletionScope.PUBLIC.",
	//       "enum": [
	//         "COMPLETION_SCOPE_UNSPECIFIED",
	//         "TENANT",
	//         "PUBLIC"
	//       ],
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "type": {
	//       "description": "Optional.\n\nThe completion topic. The default is CompletionType.COMBINED.",
	//       "enum": [
	//         "COMPLETION_TYPE_UNSPECIFIED",
	//         "JOB_TITLE",
	//         "COMPANY_NAME",
	//         "COMBINED"
	//       ],
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+name}:complete",
	//   "response": {
	//     "$ref": "CompleteQueryResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// method id "jobs.projects.companies.create":

type ProjectsCompaniesCreateCall struct {
	s                    *Service
	parent               string
	createcompanyrequest *CreateCompanyRequest
	urlParams_           gensupport.URLParams
	ctx_                 context.Context
	header_              http.Header
}

// Create: Creates a new company entity.
func (r *ProjectsCompaniesService) Create(parent string, createcompanyrequest *CreateCompanyRequest) *ProjectsCompaniesCreateCall {
	c := &ProjectsCompaniesCreateCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.parent = parent
	c.createcompanyrequest = createcompanyrequest
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsCompaniesCreateCall) Fields(s ...googleapi.Field) *ProjectsCompaniesCreateCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsCompaniesCreateCall) Context(ctx context.Context) *ProjectsCompaniesCreateCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsCompaniesCreateCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsCompaniesCreateCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.createcompanyrequest)
	if err != nil {
		return nil, err
	}
	reqHeaders.Set("Content-Type", "application/json")
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+parent}/companies")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"parent": c.parent,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.companies.create" call.
// Exactly one of *Company or error will be non-nil. Any non-2xx status
// code is an error. Response headers are in either
// *Company.ServerResponse.Header or (if a response was returned at all)
// in error.(*googleapi.Error).Header. Use googleapi.IsNotModified to
// check whether the returned error was because http.StatusNotModified
// was returned.
func (c *ProjectsCompaniesCreateCall) Do(opts ...googleapi.CallOption) (*Company, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &Company{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a new company entity.",
	//   "flatPath": "v3/projects/{projectsId}/companies",
	//   "httpMethod": "POST",
	//   "id": "jobs.projects.companies.create",
	//   "parameterOrder": [
	//     "parent"
	//   ],
	//   "parameters": {
	//     "parent": {
	//       "description": "Required.\n\nResource name of the project under which the company is created.\n\nThe format is \"projects/{project_id}\", for example,\n\"projects/api-test-project\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+parent}/companies",
	//   "request": {
	//     "$ref": "CreateCompanyRequest"
	//   },
	//   "response": {
	//     "$ref": "Company"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// method id "jobs.projects.companies.delete":

type ProjectsCompaniesDeleteCall struct {
	s          *Service
	name       string
	urlParams_ gensupport.URLParams
	ctx_       context.Context
	header_    http.Header
}

// Delete: Deletes specified company.
func (r *ProjectsCompaniesService) Delete(name string) *ProjectsCompaniesDeleteCall {
	c := &ProjectsCompaniesDeleteCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.name = name
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsCompaniesDeleteCall) Fields(s ...googleapi.Field) *ProjectsCompaniesDeleteCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsCompaniesDeleteCall) Context(ctx context.Context) *ProjectsCompaniesDeleteCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsCompaniesDeleteCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsCompaniesDeleteCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+name}")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"name": c.name,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.companies.delete" call.
// Exactly one of *Empty or error will be non-nil. Any non-2xx status
// code is an error. Response headers are in either
// *Empty.ServerResponse.Header or (if a response was returned at all)
// in error.(*googleapi.Error).Header. Use googleapi.IsNotModified to
// check whether the returned error was because http.StatusNotModified
// was returned.
func (c *ProjectsCompaniesDeleteCall) Do(opts ...googleapi.CallOption) (*Empty, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &Empty{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Deletes specified company.",
	//   "flatPath": "v3/projects/{projectsId}/companies/{companiesId}",
	//   "httpMethod": "DELETE",
	//   "id": "jobs.projects.companies.delete",
	//   "parameterOrder": [
	//     "name"
	//   ],
	//   "parameters": {
	//     "name": {
	//       "description": "Required.\n\nThe resource name of the company to be deleted.\n\nThe format is \"projects/{project_id}/companies/{company_id}\", for example,\n\"projects/api-test-project/companies/foo\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+/companies/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+name}",
	//   "response": {
	//     "$ref": "Empty"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// method id "jobs.projects.companies.get":

type ProjectsCompaniesGetCall struct {
	s            *Service
	name         string
	urlParams_   gensupport.URLParams
	ifNoneMatch_ string
	ctx_         context.Context
	header_      http.Header
}

// Get: Retrieves specified company.
func (r *ProjectsCompaniesService) Get(name string) *ProjectsCompaniesGetCall {
	c := &ProjectsCompaniesGetCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.name = name
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsCompaniesGetCall) Fields(s ...googleapi.Field) *ProjectsCompaniesGetCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// IfNoneMatch sets the optional parameter which makes the operation
// fail if the object's ETag matches the given value. This is useful for
// getting updates only after the object has changed since the last
// request. Use googleapi.IsNotModified to check whether the response
// error from Do is the result of In-None-Match.
func (c *ProjectsCompaniesGetCall) IfNoneMatch(entityTag string) *ProjectsCompaniesGetCall {
	c.ifNoneMatch_ = entityTag
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsCompaniesGetCall) Context(ctx context.Context) *ProjectsCompaniesGetCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsCompaniesGetCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsCompaniesGetCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	if c.ifNoneMatch_ != "" {
		reqHeaders.Set("If-None-Match", c.ifNoneMatch_)
	}
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+name}")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"name": c.name,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.companies.get" call.
// Exactly one of *Company or error will be non-nil. Any non-2xx status
// code is an error. Response headers are in either
// *Company.ServerResponse.Header or (if a response was returned at all)
// in error.(*googleapi.Error).Header. Use googleapi.IsNotModified to
// check whether the returned error was because http.StatusNotModified
// was returned.
func (c *ProjectsCompaniesGetCall) Do(opts ...googleapi.CallOption) (*Company, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &Company{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves specified company.",
	//   "flatPath": "v3/projects/{projectsId}/companies/{companiesId}",
	//   "httpMethod": "GET",
	//   "id": "jobs.projects.companies.get",
	//   "parameterOrder": [
	//     "name"
	//   ],
	//   "parameters": {
	//     "name": {
	//       "description": "Required.\n\nThe resource name of the company to be retrieved.\n\nThe format is \"projects/{project_id}/companies/{company_id}\", for example,\n\"projects/api-test-project/companies/foo\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+/companies/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+name}",
	//   "response": {
	//     "$ref": "Company"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// method id "jobs.projects.companies.list":

type ProjectsCompaniesListCall struct {
	s            *Service
	parent       string
	urlParams_   gensupport.URLParams
	ifNoneMatch_ string
	ctx_         context.Context
	header_      http.Header
}

// List: Lists all companies associated with the service account.
func (r *ProjectsCompaniesService) List(parent string) *ProjectsCompaniesListCall {
	c := &ProjectsCompaniesListCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.parent = parent
	return c
}

// PageSize sets the optional parameter "pageSize": The maximum number
// of companies to be returned, at most 100.
// Default is 100 if a non-positive number is provided.
func (c *ProjectsCompaniesListCall) PageSize(pageSize int64) *ProjectsCompaniesListCall {
	c.urlParams_.Set("pageSize", fmt.Sprint(pageSize))
	return c
}

// PageToken sets the optional parameter "pageToken": The starting
// indicator from which to return results.
func (c *ProjectsCompaniesListCall) PageToken(pageToken string) *ProjectsCompaniesListCall {
	c.urlParams_.Set("pageToken", pageToken)
	return c
}

// RequireOpenJobs sets the optional parameter "requireOpenJobs": Set to
// true if the companies requested must have open jobs.
//
// Defaults to false.
//
// If true, at most page_size of companies are fetched, among which
// only those with open jobs are returned.
func (c *ProjectsCompaniesListCall) RequireOpenJobs(requireOpenJobs bool) *ProjectsCompaniesListCall {
	c.urlParams_.Set("requireOpenJobs", fmt.Sprint(requireOpenJobs))
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsCompaniesListCall) Fields(s ...googleapi.Field) *ProjectsCompaniesListCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// IfNoneMatch sets the optional parameter which makes the operation
// fail if the object's ETag matches the given value. This is useful for
// getting updates only after the object has changed since the last
// request. Use googleapi.IsNotModified to check whether the response
// error from Do is the result of In-None-Match.
func (c *ProjectsCompaniesListCall) IfNoneMatch(entityTag string) *ProjectsCompaniesListCall {
	c.ifNoneMatch_ = entityTag
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsCompaniesListCall) Context(ctx context.Context) *ProjectsCompaniesListCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsCompaniesListCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsCompaniesListCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	if c.ifNoneMatch_ != "" {
		reqHeaders.Set("If-None-Match", c.ifNoneMatch_)
	}
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+parent}/companies")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"parent": c.parent,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.companies.list" call.
// Exactly one of *ListCompaniesResponse or error will be non-nil. Any
// non-2xx status code is an error. Response headers are in either
// *ListCompaniesResponse.ServerResponse.Header or (if a response was
// returned at all) in error.(*googleapi.Error).Header. Use
// googleapi.IsNotModified to check whether the returned error was
// because http.StatusNotModified was returned.
func (c *ProjectsCompaniesListCall) Do(opts ...googleapi.CallOption) (*ListCompaniesResponse, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &ListCompaniesResponse{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all companies associated with the service account.",
	//   "flatPath": "v3/projects/{projectsId}/companies",
	//   "httpMethod": "GET",
	//   "id": "jobs.projects.companies.list",
	//   "parameterOrder": [
	//     "parent"
	//   ],
	//   "parameters": {
	//     "pageSize": {
	//       "description": "Optional.\n\nThe maximum number of companies to be returned, at most 100.\nDefault is 100 if a non-positive number is provided.",
	//       "format": "int32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Optional.\n\nThe starting indicator from which to return results.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "parent": {
	//       "description": "Required.\n\nResource name of the project under which the company is created.\n\nThe format is \"projects/{project_id}\", for example,\n\"projects/api-test-project\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "requireOpenJobs": {
	//       "description": "Optional.\n\nSet to true if the companies requested must have open jobs.\n\nDefaults to false.\n\nIf true, at most page_size of companies are fetched, among which\nonly those with open jobs are returned.",
	//       "location": "query",
	//       "type": "boolean"
	//     }
	//   },
	//   "path": "v3/{+parent}/companies",
	//   "response": {
	//     "$ref": "ListCompaniesResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// Pages invokes f for each page of results.
// A non-nil error returned from f will halt the iteration.
// The provided context supersedes any context provided to the Context method.
func (c *ProjectsCompaniesListCall) Pages(ctx context.Context, f func(*ListCompaniesResponse) error) error {
	c.ctx_ = ctx
	defer c.PageToken(c.urlParams_.Get("pageToken")) // reset paging to original point
	for {
		x, err := c.Do()
		if err != nil {
			return err
		}
		if err := f(x); err != nil {
			return err
		}
		if x.NextPageToken == "" {
			return nil
		}
		c.PageToken(x.NextPageToken)
	}
}

// method id "jobs.projects.companies.patch":

type ProjectsCompaniesPatchCall struct {
	s                    *Service
	name                 string
	updatecompanyrequest *UpdateCompanyRequest
	urlParams_           gensupport.URLParams
	ctx_                 context.Context
	header_              http.Header
}

// Patch: Updates specified company. Company names can't be updated. To
// update a
// company name, delete the company and all jobs associated with it, and
// only
// then re-create them.
func (r *ProjectsCompaniesService) Patch(name string, updatecompanyrequest *UpdateCompanyRequest) *ProjectsCompaniesPatchCall {
	c := &ProjectsCompaniesPatchCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.name = name
	c.updatecompanyrequest = updatecompanyrequest
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsCompaniesPatchCall) Fields(s ...googleapi.Field) *ProjectsCompaniesPatchCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsCompaniesPatchCall) Context(ctx context.Context) *ProjectsCompaniesPatchCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsCompaniesPatchCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsCompaniesPatchCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.updatecompanyrequest)
	if err != nil {
		return nil, err
	}
	reqHeaders.Set("Content-Type", "application/json")
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+name}")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"name": c.name,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.companies.patch" call.
// Exactly one of *Company or error will be non-nil. Any non-2xx status
// code is an error. Response headers are in either
// *Company.ServerResponse.Header or (if a response was returned at all)
// in error.(*googleapi.Error).Header. Use googleapi.IsNotModified to
// check whether the returned error was because http.StatusNotModified
// was returned.
func (c *ProjectsCompaniesPatchCall) Do(opts ...googleapi.CallOption) (*Company, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &Company{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates specified company. Company names can't be updated. To update a\ncompany name, delete the company and all jobs associated with it, and only\nthen re-create them.",
	//   "flatPath": "v3/projects/{projectsId}/companies/{companiesId}",
	//   "httpMethod": "PATCH",
	//   "id": "jobs.projects.companies.patch",
	//   "parameterOrder": [
	//     "name"
	//   ],
	//   "parameters": {
	//     "name": {
	//       "description": "Required during company update.\n\nThe resource name for a company. This is generated by the service when a\ncompany is created.\n\nThe format is \"projects/{project_id}/companies/{company_id}\", for example,\n\"projects/api-test-project/companies/foo\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+/companies/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+name}",
	//   "request": {
	//     "$ref": "UpdateCompanyRequest"
	//   },
	//   "response": {
	//     "$ref": "Company"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// method id "jobs.projects.jobs.batchDelete":

type ProjectsJobsBatchDeleteCall struct {
	s                      *Service
	parent                 string
	batchdeletejobsrequest *BatchDeleteJobsRequest
	urlParams_             gensupport.URLParams
	ctx_                   context.Context
	header_                http.Header
}

// BatchDelete: Deletes a list of Jobs by filter.
func (r *ProjectsJobsService) BatchDelete(parent string, batchdeletejobsrequest *BatchDeleteJobsRequest) *ProjectsJobsBatchDeleteCall {
	c := &ProjectsJobsBatchDeleteCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.parent = parent
	c.batchdeletejobsrequest = batchdeletejobsrequest
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsJobsBatchDeleteCall) Fields(s ...googleapi.Field) *ProjectsJobsBatchDeleteCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsJobsBatchDeleteCall) Context(ctx context.Context) *ProjectsJobsBatchDeleteCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsJobsBatchDeleteCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsJobsBatchDeleteCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.batchdeletejobsrequest)
	if err != nil {
		return nil, err
	}
	reqHeaders.Set("Content-Type", "application/json")
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+parent}/jobs:batchDelete")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"parent": c.parent,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.jobs.batchDelete" call.
// Exactly one of *Empty or error will be non-nil. Any non-2xx status
// code is an error. Response headers are in either
// *Empty.ServerResponse.Header or (if a response was returned at all)
// in error.(*googleapi.Error).Header. Use googleapi.IsNotModified to
// check whether the returned error was because http.StatusNotModified
// was returned.
func (c *ProjectsJobsBatchDeleteCall) Do(opts ...googleapi.CallOption) (*Empty, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &Empty{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Deletes a list of Jobs by filter.",
	//   "flatPath": "v3/projects/{projectsId}/jobs:batchDelete",
	//   "httpMethod": "POST",
	//   "id": "jobs.projects.jobs.batchDelete",
	//   "parameterOrder": [
	//     "parent"
	//   ],
	//   "parameters": {
	//     "parent": {
	//       "description": "Required.\n\nThe resource name of the project under which the job is created.\n\nThe format is \"projects/{project_id}\", for example,\n\"projects/api-test-project\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+parent}/jobs:batchDelete",
	//   "request": {
	//     "$ref": "BatchDeleteJobsRequest"
	//   },
	//   "response": {
	//     "$ref": "Empty"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// method id "jobs.projects.jobs.create":

type ProjectsJobsCreateCall struct {
	s                *Service
	parent           string
	createjobrequest *CreateJobRequest
	urlParams_       gensupport.URLParams
	ctx_             context.Context
	header_          http.Header
}

// Create: Creates a new job.
//
// Typically, the job becomes searchable within 10 seconds, but it may
// take
// up to 5 minutes.
func (r *ProjectsJobsService) Create(parent string, createjobrequest *CreateJobRequest) *ProjectsJobsCreateCall {
	c := &ProjectsJobsCreateCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.parent = parent
	c.createjobrequest = createjobrequest
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsJobsCreateCall) Fields(s ...googleapi.Field) *ProjectsJobsCreateCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsJobsCreateCall) Context(ctx context.Context) *ProjectsJobsCreateCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsJobsCreateCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsJobsCreateCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.createjobrequest)
	if err != nil {
		return nil, err
	}
	reqHeaders.Set("Content-Type", "application/json")
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+parent}/jobs")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"parent": c.parent,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.jobs.create" call.
// Exactly one of *Job or error will be non-nil. Any non-2xx status code
// is an error. Response headers are in either
// *Job.ServerResponse.Header or (if a response was returned at all) in
// error.(*googleapi.Error).Header. Use googleapi.IsNotModified to check
// whether the returned error was because http.StatusNotModified was
// returned.
func (c *ProjectsJobsCreateCall) Do(opts ...googleapi.CallOption) (*Job, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &Job{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a new job.\n\nTypically, the job becomes searchable within 10 seconds, but it may take\nup to 5 minutes.",
	//   "flatPath": "v3/projects/{projectsId}/jobs",
	//   "httpMethod": "POST",
	//   "id": "jobs.projects.jobs.create",
	//   "parameterOrder": [
	//     "parent"
	//   ],
	//   "parameters": {
	//     "parent": {
	//       "description": "Required.\n\nThe resource name of the project under which the job is created.\n\nThe format is \"projects/{project_id}\", for example,\n\"projects/api-test-project\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+parent}/jobs",
	//   "request": {
	//     "$ref": "CreateJobRequest"
	//   },
	//   "response": {
	//     "$ref": "Job"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// method id "jobs.projects.jobs.delete":

type ProjectsJobsDeleteCall struct {
	s          *Service
	name       string
	urlParams_ gensupport.URLParams
	ctx_       context.Context
	header_    http.Header
}

// Delete: Deletes the specified job.
//
// Typically, the job becomes unsearchable within 10 seconds, but it may
// take
// up to 5 minutes.
func (r *ProjectsJobsService) Delete(name string) *ProjectsJobsDeleteCall {
	c := &ProjectsJobsDeleteCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.name = name
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsJobsDeleteCall) Fields(s ...googleapi.Field) *ProjectsJobsDeleteCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsJobsDeleteCall) Context(ctx context.Context) *ProjectsJobsDeleteCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsJobsDeleteCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsJobsDeleteCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+name}")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"name": c.name,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.jobs.delete" call.
// Exactly one of *Empty or error will be non-nil. Any non-2xx status
// code is an error. Response headers are in either
// *Empty.ServerResponse.Header or (if a response was returned at all)
// in error.(*googleapi.Error).Header. Use googleapi.IsNotModified to
// check whether the returned error was because http.StatusNotModified
// was returned.
func (c *ProjectsJobsDeleteCall) Do(opts ...googleapi.CallOption) (*Empty, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &Empty{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Deletes the specified job.\n\nTypically, the job becomes unsearchable within 10 seconds, but it may take\nup to 5 minutes.",
	//   "flatPath": "v3/projects/{projectsId}/jobs/{jobsId}",
	//   "httpMethod": "DELETE",
	//   "id": "jobs.projects.jobs.delete",
	//   "parameterOrder": [
	//     "name"
	//   ],
	//   "parameters": {
	//     "name": {
	//       "description": "Required.\n\nThe resource name of the job to be deleted.\n\nThe format is \"projects/{project_id}/jobs/{job_id}\",\nfor example, \"projects/api-test-project/jobs/1234\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+/jobs/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+name}",
	//   "response": {
	//     "$ref": "Empty"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// method id "jobs.projects.jobs.get":

type ProjectsJobsGetCall struct {
	s            *Service
	name         string
	urlParams_   gensupport.URLParams
	ifNoneMatch_ string
	ctx_         context.Context
	header_      http.Header
}

// Get: Retrieves the specified job, whose status is OPEN or recently
// EXPIRED
// within the last 90 days.
func (r *ProjectsJobsService) Get(name string) *ProjectsJobsGetCall {
	c := &ProjectsJobsGetCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.name = name
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsJobsGetCall) Fields(s ...googleapi.Field) *ProjectsJobsGetCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// IfNoneMatch sets the optional parameter which makes the operation
// fail if the object's ETag matches the given value. This is useful for
// getting updates only after the object has changed since the last
// request. Use googleapi.IsNotModified to check whether the response
// error from Do is the result of In-None-Match.
func (c *ProjectsJobsGetCall) IfNoneMatch(entityTag string) *ProjectsJobsGetCall {
	c.ifNoneMatch_ = entityTag
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsJobsGetCall) Context(ctx context.Context) *ProjectsJobsGetCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsJobsGetCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsJobsGetCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	if c.ifNoneMatch_ != "" {
		reqHeaders.Set("If-None-Match", c.ifNoneMatch_)
	}
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+name}")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"name": c.name,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.jobs.get" call.
// Exactly one of *Job or error will be non-nil. Any non-2xx status code
// is an error. Response headers are in either
// *Job.ServerResponse.Header or (if a response was returned at all) in
// error.(*googleapi.Error).Header. Use googleapi.IsNotModified to check
// whether the returned error was because http.StatusNotModified was
// returned.
func (c *ProjectsJobsGetCall) Do(opts ...googleapi.CallOption) (*Job, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &Job{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves the specified job, whose status is OPEN or recently EXPIRED\nwithin the last 90 days.",
	//   "flatPath": "v3/projects/{projectsId}/jobs/{jobsId}",
	//   "httpMethod": "GET",
	//   "id": "jobs.projects.jobs.get",
	//   "parameterOrder": [
	//     "name"
	//   ],
	//   "parameters": {
	//     "name": {
	//       "description": "Required.\n\nThe resource name of the job to retrieve.\n\nThe format is \"projects/{project_id}/jobs/{job_id}\",\nfor example, \"projects/api-test-project/jobs/1234\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+/jobs/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+name}",
	//   "response": {
	//     "$ref": "Job"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// method id "jobs.projects.jobs.list":

type ProjectsJobsListCall struct {
	s            *Service
	parent       string
	urlParams_   gensupport.URLParams
	ifNoneMatch_ string
	ctx_         context.Context
	header_      http.Header
}

// List: Lists jobs by filter.
func (r *ProjectsJobsService) List(parent string) *ProjectsJobsListCall {
	c := &ProjectsJobsListCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.parent = parent
	return c
}

// Filter sets the optional parameter "filter": Required.
//
// The filter string specifies the jobs to be enumerated.
//
// Supported operator: =, AND
//
// The fields eligible for filtering are:
//
// * `companyName` (Required)
// * `requisitionId` (Optional)
//
// Sample Query:
//
// * companyName = "projects/api-test-project/companies/123"
// * companyName = "projects/api-test-project/companies/123" AND
// requisitionId
// = "req-1"
func (c *ProjectsJobsListCall) Filter(filter string) *ProjectsJobsListCall {
	c.urlParams_.Set("filter", filter)
	return c
}

// JobView sets the optional parameter "jobView": The desired job
// attributes returned for jobs in the
// search response. Defaults to JobView.JOB_VIEW_FULL if no value
// is
// specified.
//
// Possible values:
//   "JOB_VIEW_UNSPECIFIED"
//   "JOB_VIEW_ID_ONLY"
//   "JOB_VIEW_MINIMAL"
//   "JOB_VIEW_SMALL"
//   "JOB_VIEW_FULL"
func (c *ProjectsJobsListCall) JobView(jobView string) *ProjectsJobsListCall {
	c.urlParams_.Set("jobView", jobView)
	return c
}

// PageSize sets the optional parameter "pageSize": The maximum number
// of jobs to be returned per page of results.
//
// If job_view is set to JobView.JOB_VIEW_ID_ONLY, the maximum
// allowed
// page size is 1000. Otherwise, the maximum allowed page size is
// 100.
//
// Default is 100 if empty or a number < 1 is specified.
func (c *ProjectsJobsListCall) PageSize(pageSize int64) *ProjectsJobsListCall {
	c.urlParams_.Set("pageSize", fmt.Sprint(pageSize))
	return c
}

// PageToken sets the optional parameter "pageToken": The starting point
// of a query result.
func (c *ProjectsJobsListCall) PageToken(pageToken string) *ProjectsJobsListCall {
	c.urlParams_.Set("pageToken", pageToken)
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsJobsListCall) Fields(s ...googleapi.Field) *ProjectsJobsListCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// IfNoneMatch sets the optional parameter which makes the operation
// fail if the object's ETag matches the given value. This is useful for
// getting updates only after the object has changed since the last
// request. Use googleapi.IsNotModified to check whether the response
// error from Do is the result of In-None-Match.
func (c *ProjectsJobsListCall) IfNoneMatch(entityTag string) *ProjectsJobsListCall {
	c.ifNoneMatch_ = entityTag
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsJobsListCall) Context(ctx context.Context) *ProjectsJobsListCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsJobsListCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsJobsListCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	if c.ifNoneMatch_ != "" {
		reqHeaders.Set("If-None-Match", c.ifNoneMatch_)
	}
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+parent}/jobs")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"parent": c.parent,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.jobs.list" call.
// Exactly one of *ListJobsResponse or error will be non-nil. Any
// non-2xx status code is an error. Response headers are in either
// *ListJobsResponse.ServerResponse.Header or (if a response was
// returned at all) in error.(*googleapi.Error).Header. Use
// googleapi.IsNotModified to check whether the returned error was
// because http.StatusNotModified was returned.
func (c *ProjectsJobsListCall) Do(opts ...googleapi.CallOption) (*ListJobsResponse, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &ListJobsResponse{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists jobs by filter.",
	//   "flatPath": "v3/projects/{projectsId}/jobs",
	//   "httpMethod": "GET",
	//   "id": "jobs.projects.jobs.list",
	//   "parameterOrder": [
	//     "parent"
	//   ],
	//   "parameters": {
	//     "filter": {
	//       "description": "Required.\n\nThe filter string specifies the jobs to be enumerated.\n\nSupported operator: =, AND\n\nThe fields eligible for filtering are:\n\n* `companyName` (Required)\n* `requisitionId` (Optional)\n\nSample Query:\n\n* companyName = \"projects/api-test-project/companies/123\"\n* companyName = \"projects/api-test-project/companies/123\" AND requisitionId\n= \"req-1\"",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "jobView": {
	//       "description": "Optional.\n\nThe desired job attributes returned for jobs in the\nsearch response. Defaults to JobView.JOB_VIEW_FULL if no value is\nspecified.",
	//       "enum": [
	//         "JOB_VIEW_UNSPECIFIED",
	//         "JOB_VIEW_ID_ONLY",
	//         "JOB_VIEW_MINIMAL",
	//         "JOB_VIEW_SMALL",
	//         "JOB_VIEW_FULL"
	//       ],
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "pageSize": {
	//       "description": "Optional.\n\nThe maximum number of jobs to be returned per page of results.\n\nIf job_view is set to JobView.JOB_VIEW_ID_ONLY, the maximum allowed\npage size is 1000. Otherwise, the maximum allowed page size is 100.\n\nDefault is 100 if empty or a number \u003c 1 is specified.",
	//       "format": "int32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Optional.\n\nThe starting point of a query result.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "parent": {
	//       "description": "Required.\n\nThe resource name of the project under which the job is created.\n\nThe format is \"projects/{project_id}\", for example,\n\"projects/api-test-project\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+parent}/jobs",
	//   "response": {
	//     "$ref": "ListJobsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// Pages invokes f for each page of results.
// A non-nil error returned from f will halt the iteration.
// The provided context supersedes any context provided to the Context method.
func (c *ProjectsJobsListCall) Pages(ctx context.Context, f func(*ListJobsResponse) error) error {
	c.ctx_ = ctx
	defer c.PageToken(c.urlParams_.Get("pageToken")) // reset paging to original point
	for {
		x, err := c.Do()
		if err != nil {
			return err
		}
		if err := f(x); err != nil {
			return err
		}
		if x.NextPageToken == "" {
			return nil
		}
		c.PageToken(x.NextPageToken)
	}
}

// method id "jobs.projects.jobs.patch":

type ProjectsJobsPatchCall struct {
	s                *Service
	name             string
	updatejobrequest *UpdateJobRequest
	urlParams_       gensupport.URLParams
	ctx_             context.Context
	header_          http.Header
}

// Patch: Updates specified job.
//
// Typically, updated contents become visible in search results within
// 10
// seconds, but it may take up to 5 minutes.
func (r *ProjectsJobsService) Patch(name string, updatejobrequest *UpdateJobRequest) *ProjectsJobsPatchCall {
	c := &ProjectsJobsPatchCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.name = name
	c.updatejobrequest = updatejobrequest
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsJobsPatchCall) Fields(s ...googleapi.Field) *ProjectsJobsPatchCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsJobsPatchCall) Context(ctx context.Context) *ProjectsJobsPatchCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsJobsPatchCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsJobsPatchCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.updatejobrequest)
	if err != nil {
		return nil, err
	}
	reqHeaders.Set("Content-Type", "application/json")
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+name}")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"name": c.name,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.jobs.patch" call.
// Exactly one of *Job or error will be non-nil. Any non-2xx status code
// is an error. Response headers are in either
// *Job.ServerResponse.Header or (if a response was returned at all) in
// error.(*googleapi.Error).Header. Use googleapi.IsNotModified to check
// whether the returned error was because http.StatusNotModified was
// returned.
func (c *ProjectsJobsPatchCall) Do(opts ...googleapi.CallOption) (*Job, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &Job{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates specified job.\n\nTypically, updated contents become visible in search results within 10\nseconds, but it may take up to 5 minutes.",
	//   "flatPath": "v3/projects/{projectsId}/jobs/{jobsId}",
	//   "httpMethod": "PATCH",
	//   "id": "jobs.projects.jobs.patch",
	//   "parameterOrder": [
	//     "name"
	//   ],
	//   "parameters": {
	//     "name": {
	//       "description": "Required during job update.\n\nThe resource name for the job. This is generated by the service when a\njob is created.\n\nThe format is \"projects/{project_id}/jobs/{job_id}\",\nfor example, \"projects/api-test-project/jobs/1234\".\n\nUse of this field in job queries and API calls is preferred over the use of\nrequisition_id since this value is unique.",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+/jobs/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+name}",
	//   "request": {
	//     "$ref": "UpdateJobRequest"
	//   },
	//   "response": {
	//     "$ref": "Job"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// method id "jobs.projects.jobs.search":

type ProjectsJobsSearchCall struct {
	s                 *Service
	parent            string
	searchjobsrequest *SearchJobsRequest
	urlParams_        gensupport.URLParams
	ctx_              context.Context
	header_           http.Header
}

// Search: Searches for jobs using the provided SearchJobsRequest.
//
// This call constrains the visibility of jobs
// present in the database, and only returns jobs that the caller
// has
// permission to search against.
func (r *ProjectsJobsService) Search(parent string, searchjobsrequest *SearchJobsRequest) *ProjectsJobsSearchCall {
	c := &ProjectsJobsSearchCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.parent = parent
	c.searchjobsrequest = searchjobsrequest
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsJobsSearchCall) Fields(s ...googleapi.Field) *ProjectsJobsSearchCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsJobsSearchCall) Context(ctx context.Context) *ProjectsJobsSearchCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsJobsSearchCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsJobsSearchCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.searchjobsrequest)
	if err != nil {
		return nil, err
	}
	reqHeaders.Set("Content-Type", "application/json")
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+parent}/jobs:search")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"parent": c.parent,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.jobs.search" call.
// Exactly one of *SearchJobsResponse or error will be non-nil. Any
// non-2xx status code is an error. Response headers are in either
// *SearchJobsResponse.ServerResponse.Header or (if a response was
// returned at all) in error.(*googleapi.Error).Header. Use
// googleapi.IsNotModified to check whether the returned error was
// because http.StatusNotModified was returned.
func (c *ProjectsJobsSearchCall) Do(opts ...googleapi.CallOption) (*SearchJobsResponse, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &SearchJobsResponse{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Searches for jobs using the provided SearchJobsRequest.\n\nThis call constrains the visibility of jobs\npresent in the database, and only returns jobs that the caller has\npermission to search against.",
	//   "flatPath": "v3/projects/{projectsId}/jobs:search",
	//   "httpMethod": "POST",
	//   "id": "jobs.projects.jobs.search",
	//   "parameterOrder": [
	//     "parent"
	//   ],
	//   "parameters": {
	//     "parent": {
	//       "description": "Required.\n\nThe resource name of the project to search within.\n\nThe format is \"projects/{project_id}\", for example,\n\"projects/api-test-project\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+parent}/jobs:search",
	//   "request": {
	//     "$ref": "SearchJobsRequest"
	//   },
	//   "response": {
	//     "$ref": "SearchJobsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// Pages invokes f for each page of results.
// A non-nil error returned from f will halt the iteration.
// The provided context supersedes any context provided to the Context method.
func (c *ProjectsJobsSearchCall) Pages(ctx context.Context, f func(*SearchJobsResponse) error) error {
	c.ctx_ = ctx
	defer func(pt string) { c.searchjobsrequest.PageToken = pt }(c.searchjobsrequest.PageToken) // reset paging to original point
	for {
		x, err := c.Do()
		if err != nil {
			return err
		}
		if err := f(x); err != nil {
			return err
		}
		if x.NextPageToken == "" {
			return nil
		}
		c.searchjobsrequest.PageToken = x.NextPageToken
	}
}

// method id "jobs.projects.jobs.searchForAlert":

type ProjectsJobsSearchForAlertCall struct {
	s                 *Service
	parent            string
	searchjobsrequest *SearchJobsRequest
	urlParams_        gensupport.URLParams
	ctx_              context.Context
	header_           http.Header
}

// SearchForAlert: Searches for jobs using the provided
// SearchJobsRequest.
//
// This API call is intended for the use case of targeting passive
// job
// seekers (for example, job seekers who have signed up to receive
// email
// alerts about potential job opportunities), and has different
// algorithmic
// adjustments that are targeted to passive job seekers.
//
// This call constrains the visibility of jobs
// present in the database, and only returns jobs the caller
// has
// permission to search against.
func (r *ProjectsJobsService) SearchForAlert(parent string, searchjobsrequest *SearchJobsRequest) *ProjectsJobsSearchForAlertCall {
	c := &ProjectsJobsSearchForAlertCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.parent = parent
	c.searchjobsrequest = searchjobsrequest
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsJobsSearchForAlertCall) Fields(s ...googleapi.Field) *ProjectsJobsSearchForAlertCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsJobsSearchForAlertCall) Context(ctx context.Context) *ProjectsJobsSearchForAlertCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsJobsSearchForAlertCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsJobsSearchForAlertCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.searchjobsrequest)
	if err != nil {
		return nil, err
	}
	reqHeaders.Set("Content-Type", "application/json")
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v3/{+parent}/jobs:searchForAlert")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"parent": c.parent,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "jobs.projects.jobs.searchForAlert" call.
// Exactly one of *SearchJobsResponse or error will be non-nil. Any
// non-2xx status code is an error. Response headers are in either
// *SearchJobsResponse.ServerResponse.Header or (if a response was
// returned at all) in error.(*googleapi.Error).Header. Use
// googleapi.IsNotModified to check whether the returned error was
// because http.StatusNotModified was returned.
func (c *ProjectsJobsSearchForAlertCall) Do(opts ...googleapi.CallOption) (*SearchJobsResponse, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &SearchJobsResponse{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Searches for jobs using the provided SearchJobsRequest.\n\nThis API call is intended for the use case of targeting passive job\nseekers (for example, job seekers who have signed up to receive email\nalerts about potential job opportunities), and has different algorithmic\nadjustments that are targeted to passive job seekers.\n\nThis call constrains the visibility of jobs\npresent in the database, and only returns jobs the caller has\npermission to search against.",
	//   "flatPath": "v3/projects/{projectsId}/jobs:searchForAlert",
	//   "httpMethod": "POST",
	//   "id": "jobs.projects.jobs.searchForAlert",
	//   "parameterOrder": [
	//     "parent"
	//   ],
	//   "parameters": {
	//     "parent": {
	//       "description": "Required.\n\nThe resource name of the project to search within.\n\nThe format is \"projects/{project_id}\", for example,\n\"projects/api-test-project\".",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v3/{+parent}/jobs:searchForAlert",
	//   "request": {
	//     "$ref": "SearchJobsRequest"
	//   },
	//   "response": {
	//     "$ref": "SearchJobsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/jobs"
	//   ]
	// }

}

// Pages invokes f for each page of results.
// A non-nil error returned from f will halt the iteration.
// The provided context supersedes any context provided to the Context method.
func (c *ProjectsJobsSearchForAlertCall) Pages(ctx context.Context, f func(*SearchJobsResponse) error) error {
	c.ctx_ = ctx
	defer func(pt string) { c.searchjobsrequest.PageToken = pt }(c.searchjobsrequest.PageToken) // reset paging to original point
	for {
		x, err := c.Do()
		if err != nil {
			return err
		}
		if err := f(x); err != nil {
			return err
		}
		if x.NextPageToken == "" {
			return nil
		}
		c.searchjobsrequest.PageToken = x.NextPageToken
	}
}
