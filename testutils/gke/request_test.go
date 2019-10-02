package gke

import "testing"

func TestNewCreateClusterRequest(t *testing.T) {
	datas := []struct {
		req           *Request
		errorExpected bool
	}{{
		req: &Request{
			GKEVersion:  "1-2-3",
			ClusterName: "name-a",
			MinNodes:    1,
			MaxNodes:    1,
			NodeType:    "n1-standard-4",
		},
		errorExpected: true,
	}, {
		req: &Request{
			Project:     "project-b",
			ClusterName: "name-b",
			MinNodes:    1,
			MaxNodes:    1,
			NodeType:    "n1-standard-4",
		},
		errorExpected: false,
	}, {
		req: &Request{
			Project:    "project-c",
			GKEVersion: "1-2-3",
			MinNodes:   1,
			MaxNodes:   1,
			NodeType:   "n1-standard-4",
		},
		errorExpected: true,
	}, {
		req: &Request{
			Project:     "project-d",
			GKEVersion:  "1-2-3",
			ClusterName: "name-d",
			MinNodes:    0,
			MaxNodes:    1,
			NodeType:    "n1-standard-4",
		},
		errorExpected: true,
	}, {
		req: &Request{
			Project:     "project-e",
			GKEVersion:  "1-2-3",
			ClusterName: "name-e",
			MinNodes:    10,
			MaxNodes:    1,
			NodeType:    "n1-standard-4",
		},
		errorExpected: true,
	}, {
		req: &Request{
			Project:     "project-f",
			GKEVersion:  "1-2-3",
			ClusterName: "name-f",
			MinNodes:    1,
			MaxNodes:    1,
		},
		errorExpected: true,
	}}
	for _, data := range datas {
		createReq, err := NewCreateClusterRequest(data.req)
		if data.errorExpected {
			if err == nil {
				t.Errorf("Expected error from request '%v', but got nil", data.req)
			}
		} else {
			if err != nil {
				t.Errorf("Expected no error from request '%v', but got '%v'", data.req, err)
			}
			if createReq == nil {
				t.Error("Expected a valid request, but got nil")
			}
		}
	}
}
