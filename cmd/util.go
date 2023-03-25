package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aundis/meta"
	"github.com/aundis/srpc"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gclient"
)

func requestObjectMeta(ctx context.Context, client *srpc.Client, target string, req helperListReq) ([]meta.ObjectMeta, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	res, err := client.Request(ctx, srpc.RequestData{
		Mark:   srpc.CallMark,
		Target: target,
		Action: "Helper.list",
		Data:   data,
	})
	if err != nil {
		return nil, err
	}
	var out *helperListRes
	err = json.Unmarshal(res, &out)
	if err != nil {
		return nil, err
	}
	return out.List, nil
}

func newSrpcClinet(ctx context.Context) (*srpc.Client, error) {
	addr, err := readConfig(ctx)
	if err != nil {
		return nil, err
	}
	name := "srpc-cli"
	client := gclient.NewWebSocket()
	conn, _, err := client.Dial(addr+fmt.Sprintf("?name=%s", name), http.Header{})
	if err != nil {
		return nil, err
	}
	sclient := srpc.NewClient(name, conn)
	go sclient.Start(ctx)
	return sclient, nil
}

func readConfig(ctx context.Context) (addr string, err error) {
	addrValue, err := g.Cfg().Get(ctx, "srpc.address")
	if err != nil {
		err = errors.New("read srpc address error: " + err.Error())
		return
	}
	addr = addrValue.String()
	return
}
