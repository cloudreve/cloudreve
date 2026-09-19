package admin

import (
	"encoding/gob"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

type ListShareResponse struct {
	Pagination *inventory.PaginationResults `json:"pagination"`
	Shares     []GetShareResponse           `json:"shares"`
}

type GetShareResponse struct {
	*ent.Share
	UserHashID string `json:"user_hash_id,omitempty"`
	ShareLink  string `json:"share_link,omitempty"`
}

type ListTaskResponse struct {
	Pagination *inventory.PaginationResults `json:"pagination"`
	Tasks      []GetTaskResponse            `json:"tasks"`
}

type GetTaskResponse struct {
	*ent.Task
	UserHashID string         `json:"user_hash_id,omitempty"`
	TaskHashID string         `json:"task_hash_id,omitempty"`
	Summary    *queue.Summary `json:"summary,omitempty"`
	Node       *ent.Node      `json:"node,omitempty"`
}

type ListEntityResponse struct {
	Pagination *inventory.PaginationResults `json:"pagination"`
	Entities   []GetEntityResponse          `json:"entities"`
}

type GetEntityResponse struct {
	*ent.Entity
	UserHashID    string         `json:"user_hash_id,omitempty"`
	UserHashIDMap map[int]string `json:"user_hash_id_map,omitempty"`
}

type ListFileResponse struct {
	Pagination *inventory.PaginationResults `json:"pagination"`
	Files      []GetFileResponse            `json:"files"`
}

type GetFileResponse struct {
	*ent.File
	UserHashID    string         `json:"user_hash_id,omitempty"`
	FileHashID    string         `json:"file_hash_id,omitempty"`
	DirectLinkMap map[int]string `json:"direct_link_map,omitempty"`
}

type ListUserResponse struct {
	Pagination *inventory.PaginationResults `json:"pagination"`
	Users      []GetUserResponse            `json:"users"`
}

type GetUserResponse struct {
	*ent.User
	HashID       string       `json:"hash_id,omitempty"`
	TwoFAEnabled bool         `json:"two_fa_enabled,omitempty"`
	Capacity     *fs.Capacity `json:"capacity,omitempty"`
}

type GetNodeResponse struct {
	*ent.Node
}

type GetGroupResponse struct {
	*ent.Group
	TotalUsers int `json:"total_users"`
}

type OauthCredentialStatus struct {
	Valid           bool       `json:"valid"`
	LastRefreshTime *time.Time `json:"last_refresh_time"`
}

type GetStoragePolicyResponse struct {
	*ent.StoragePolicy
	EntitiesCount int `json:"entities_count,omitempty"`
	EntitiesSize  int `json:"entities_size,omitempty"`
}

type ListNodeResponse struct {
	Pagination *inventory.PaginationResults `json:"pagination"`
	Nodes      []*ent.Node                  `json:"nodes"`
}

type ListPolicyResponse struct {
	Pagination *inventory.PaginationResults `json:"pagination"`
	Policies   []*ent.StoragePolicy         `json:"policies"`
}

type QueueMetric struct {
	Name            setting.QueueType `json:"name"`
	BusyWorkers     int               `json:"busy_workers"`
	SuccessTasks    int               `json:"success_tasks"`
	FailureTasks    int               `json:"failure_tasks"`
	SubmittedTasks  int               `json:"submitted_tasks"`
	SuspendingTasks int               `json:"suspending_tasks"`
}

type ListGroupResponse struct {
	Groups     []*ent.Group                 `json:"groups"`
	Pagination *inventory.PaginationResults `json:"pagination"`
}

type ListInvitationCodeResult struct {
	Pagination *inventory.PaginationResults `json:"pagination"`
	Codes      []*InvitationCodeResponse    `json:"codes"`
}

type ListOAuthClientResponse struct {
	Clients    []GetOAuthClientResponse     `json:"clients"`
	Pagination *inventory.PaginationResults `json:"pagination"`
}

type GetOAuthClientResponse struct {
	*ent.OAuthClient
	IsSystem    bool `json:"is_system"`
	TotalGrants int  `json:"total_grants,omitempty"`
}

type HomepageSummary struct {
	MetricsSummary *MetricsSummary `json:"metrics_summary"`
	SiteURls       []string        `json:"site_urls"`
	Version        *Version        `json:"version"`
}

type MetricsSummary struct {
	Dates         []time.Time `json:"dates"`
	Files         []int       `json:"files"`
	Users         []int       `json:"users"`
	Shares        []int       `json:"shares"`
	FileTotal     int         `json:"file_total"`
	UserTotal     int         `json:"user_total"`
	ShareTotal    int         `json:"share_total"`
	EntitiesTotal int         `json:"entities_total"`
	GeneratedAt   time.Time   `json:"generated_at"`
}

type Version struct {
	Version string `json:"version"`
	Pro     bool   `json:"pro"`
	Commit  string `json:"commit"`
}

func init() {
	gob.Register(MetricsSummary{})
}
