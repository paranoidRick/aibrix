/*
Copyright 2025 The Aibrix Team.

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

package api_server

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/vllm-project/aibrix/pkg/cache"
	"github.com/vllm-project/aibrix/pkg/metrics"
	"github.com/vllm-project/aibrix/pkg/types"
	"github.com/vllm-project/aibrix/pkg/utils"
	"k8s.io/klog/v2"

	routingalgorithms "github.com/vllm-project/aibrix/pkg/plugins/gateway/algorithms"
)

const RouterLeastRequest types.RoutingAlgorithm = "api-server-least-request"

func init() {
	routingalgorithms.Register(RouterLeastRequest, NewApiServerLeastRequestRouter)
}

type leastRequestRouter struct {
	cache cache.Cache
}

func NewApiServerLeastRequestRouter() (types.Router, error) {
	c, err := cache.Get()
	if err != nil {
		return nil, err
	}

	return leastRequestRouter{
		cache: c,
	}, nil
}

// Route request based of least active request among input ready pods
func (r leastRequestRouter) Route(ctx *types.RoutingContext, readyPodList types.PodList) (string, error) {
	readyPods := readyPodList.All()
	var readyApiServers []*utils.ApiServer
	for _, pod := range readyPods {
		for _, port := range utils.GetPodPorts(pod) {
			readyApiServers = append(readyApiServers, &utils.ApiServer{Pod: pod, Port: port})
		}
	}

	targetServer := selectTargetApiServerWithLeastRequestCount(r.cache, readyApiServers)

	// Use fallback if no valid metrics
	if targetServer == nil {
		var err error
		targetServer, err = routingalgorithms.SelectRandomApiServerAsFallback(ctx, readyApiServers, rand.Intn)
		if err != nil {
			return "", err
		}
	}

	ctx.SetTargetPod(targetServer.Pod)
	ctx.SetTargetPort(targetServer.Port)
	return ctx.TargetAddress(), nil
}

func (r *leastRequestRouter) SubscribedMetrics() []string {
	return []string{
		metrics.RealtimeNumRequestsRunning,
	}
}

func selectTargetApiServerWithLeastRequestCount(cache cache.Cache, readyServers []*utils.ApiServer) *utils.ApiServer {
	var targetApiServer *utils.ApiServer
	targetServers := []string{}

	minCount := math.MaxInt32
	srvRequestCount := getRequestCounts(cache, readyServers)
	klog.V(4).InfoS("selectTargetApiServerWithLeastRequestCount", "srvRequestCount", srvRequestCount)
	for servername, totalReq := range srvRequestCount {
		if totalReq < minCount {
			minCount = totalReq
			targetServers = []string{servername}
		} else if totalReq == minCount {
			targetServers = append(targetServers, servername)
		}
	}
	if len(targetServers) > 0 {
		targetApiServer, _ = utils.FilterApiServerByName(targetServers[rand.Intn(len(targetServers))], readyServers)
	}
	return targetApiServer
}

// getRequestCounts returns running request count for each pod tracked by gateway.
// Note: Currently, gateway instance tracks active running request counts for each pod locally,
// if multiple gateway instances are active then state is not shared across them.
// It is advised to run on leader gateway instance.
// TODO: Support stateful information sync across gateway instances: https://github.com/vllm-project/aibrix/issues/761
func getRequestCounts(cache cache.Cache, readyServers []*utils.ApiServer) map[string]int {
	podRequestCount := map[string]int{}
	for _, server := range readyServers {
		runningReq, err := cache.GetMetricValueByPodWithPort(server.Name, server.Namespace, metrics.RealtimeNumRequestsRunning, server.Port)
		if err != nil {
			runningReq = &metrics.SimpleMetricValue{Value: 0}
		}
		keyName := fmt.Sprintf("%s/%s/%v", server.Name, server.Namespace, server.Port)
		podRequestCount[keyName] = int(runningReq.GetSimpleValue())
	}

	return podRequestCount
}
