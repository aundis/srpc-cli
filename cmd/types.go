package cmd

import "github.com/aundis/meta"

type helperListReq struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type helperListRes struct {
	List []meta.ObjectMeta `json:"list"`
}
