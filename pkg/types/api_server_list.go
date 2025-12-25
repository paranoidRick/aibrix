package types

import "github.com/vllm-project/aibrix/pkg/utils"

type ApiServerList interface {
	Len() int

	All() []*utils.ApiServer

	Indexes() []string

	ListByIndex(index string) []*utils.ApiServer
}
