package cmd

import "github.com/aundis/mate"

type helperListReq struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type helperListRes struct {
	List []mate.ObjectMate `json:"list"`
}
