package handler

import "context"

type StatusResp struct {
	Status string `json:"status"`
}

func Status(ctx context.Context) (StatusResp, error) {
	return StatusResp{Status: "ok"}, nil
}
