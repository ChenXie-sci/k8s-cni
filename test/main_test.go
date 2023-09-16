package test

import (
	"fmt"
	"net"

	"testing"

	"encoding/json"
	"testcni/ipam"
	"testcni/nettools"

	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
)

type PluginConf struct {
	// NetConf 里头指定了一个 plugin 的最基本的信息, 比如 CNIVersion, Name, Type 等, 当然还有在 containerd 中塞进来的 PrevResult
	types.NetConf

	// 这个 runtimeConfig 是可以在 /etc/cni/net.d/xxx.conf 中配置一个
	// 类似 "capabilities": {"xxx": true, "yyy": false} 这样的属性
	// 表示说要在运行时开启 xxx 的能力, 不开启 yyy 的能力
	// 然后等容器跑起来之后(或者被拉起来之前)可以直接通过设置环境变量 export CAP_ARGS='{ "xxx": "aaaa", "yyy": "bbbb" }'
	// 来开启或关闭某些能力
	// 然后通过 stdin 标准输入读进来的数据中就会多出一个 RuntimeConfig 属性, 里面就是 runtimeConfig: { "xxx": "aaaa" }
	// 因为 yyy 在 /etc/cni/net.d/xxx.conf 中被设置为了 false
	// 官方使用范例: https://kubernetes.feisky.xyz/extension/network/cni
	// cni 源码中实现: /cni/libcni/api.go:injectRuntimeConfig
	RuntimeConfig *struct {
		TestConfig map[string]interface{} `json:"testConfig"`
	} `json:"runtimeConfig"`

	// 这里可以自由定义自己的 plugin 中配置了的参数然后自由处理
	Bridge string `json:"bridge"`
	Subnet string `json:"subnet"`
}

func TestMain(t *testing.T) {
	// 测试代码执行后, 可通过执行 ./clear.sh testcni0 来清掉测试的操作,
	// 不过注意不同节点上要把对应的其他节点 ip 改咯

	type TestArgs struct {
		ContainerID string
		Netns       string
		IfName      string
		Args        string
		Path        string
		StdinData   []byte
	}

	args := &TestArgs{
		ContainerID: "30226eb5c94ec805ecd37382c41bbac0b9ba92a8fb771c421cac856660dc6e7b",
		Netns:       "/run/netns/test.net.1",
		IfName:      "eth0",
		Args:        "",
		Path:        "/opt/cni/bin",
		StdinData:   []byte("{\"bridge\":\"testcni0\",\"capabilities\":{\"test1\":true,\"test2\":false},\"cniVersion\":\"0.3.0\",\"name\":\"testcni\",\"subnet\":\"10.244.0.0/16\",\"type\":\"testcni\"}"),
	}

	pluginConfig := &PluginConf{}
	if err := json.Unmarshal(args.StdinData, pluginConfig); err != nil {
		fmt.Println("args.StdinData 转 pluginConfig 失败: ", err.Error())
		return
	}

	fmt.Println("这里的结果是: pluginConfig.Bridge", pluginConfig.Bridge)
	fmt.Println("这里的结果是: pluginConfig.CNIVersion", pluginConfig.CNIVersion)
	fmt.Println("这里的结果是: pluginConfig.Name", pluginConfig.Name)
	fmt.Println("这里的结果是: pluginConfig.Subnet", pluginConfig.Subnet)
	fmt.Println("这里的结果是: pluginConfig.Type", pluginConfig.Type)

	// 使用 kubelet(containerd) 传过来的 subnet 地址初始化 ipam
	ipam.Init(pluginConfig.Subnet, nil)
	ipamClient, err := ipam.GetIpamService()
	if err != nil {
		fmt.Println("创建 ipam 客户端出错, err: ", err.Error())
		return
	}

	// 根据 subnet 网段来得到网关, 表示所有的节点上的 pod 的 ip 都在这个网关范围内
	gateway, err := ipamClient.Get().Gateway()
	if err != nil {
		fmt.Println("获取 gateway 出错, err: ", err.Error())
		return
	}
	fmt.Println("这里获取到的 gateway 是: ", gateway)

	// 获取网关＋网段号
	gatewayWithMaskSegment, err := ipamClient.Get().GatewayWithMaskSegment()
	if err != nil {
		fmt.Println("获取 gatewayWithMaskSegment 出错, err: ", err.Error())
		return
	}
	fmt.Println("这里获取到的 gatewayWithMaskSegment 是: ", gatewayWithMaskSegment)

	// 获取网桥名字
	bridgeName := pluginConfig.Bridge
	if bridgeName != "" {
		bridgeName = "testcni0"
	}
	// 这里如果不同节点间通信的方式使用 vxlan 的话, 这里需要变成 1460
	// 因为 vxlan 设备会给报头中加一个 40 字节的 vxlan 头部
	mtu := 1500
	// 获取 containerd 传过来的网卡名, 这个网卡名要被插到 net ns 中
	ifName := args.IfName
	// 根据 containerd 传过来的 netns 的地址获取 ns
	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		fmt.Println("获取 ns 失败: ", err.Error())
		return
	}
	// 从 ipam 中拿到一个未使用的 ip 地址
	podIP, err := ipamClient.Get().UnusedIP()
	if err != nil {
		fmt.Println("获取 podIP 出错, err: ", err.Error())
		return
	}
	// 走到这儿的话说明这个 podIP 已经在 etcd 中占上坑位了
	// 占坑的操作是直接在 Get().UnusedIP() 的时候就做了
	// 后续如果有什么 error 的话可以再 release
	fmt.Println("这里获取到的 podIP 是: ", podIP)

	// 这里拼接 pod 的 cidr
	// podIP = podIP + "/" + ipamClient.MaskSegment
	podIP = podIP + "/" + "24"
	fmt.Println("这里获取到的带有网段号的 podIP 是: ", podIP)

	/**
	 * 准备操作做完之后就可以调用网络工具来创建网络了
	 * nettools 主要做的事情:
	 *		1. 根据网桥名创建一个网桥
	 *		2. 根据网卡名儿创建一对儿 veth
	 *		3. 把叫做 IfName 的怼到 pod(netns) 上
	 *		4. 把另外一个干到主机的网桥上
	 *		5. set up 网桥以及这对儿 veth
	 *		6. 在 pod(netns) 里创建一个默认路由, 把匹配到 0.0.0.0 的 ip 都让其从 IfName 那块儿 veth 往外走
	 *		7. 设置主机的 iptables, 让所有来自 bridgeName 的流量都能做 forward(因为 docker 可能会自己设置 iptables 不让转发的规则)
	 */

	err = nettools.CreateBridgeAndCreateVethAndSetNetworkDeviceStatusAndSetVethMaster(bridgeName, gatewayWithMaskSegment, ifName, podIP, mtu, netns)
	if err != nil {
		fmt.Println("执行创建网桥, 创建 veth 设备, 添加默认路由等操作失败, err: ", err.Error())
		err = ipamClient.Release().IPs(podIP)
		if err != nil {
			fmt.Println("释放 podIP", podIP, " 失败: ", err.Error())
		}
	}

	/**
	 * 到这儿为止, 同一台主机上的 pod 可以 ping 通了
	 * 并且也可以访问其他网段的 ip 了
	 * 不过此时只能 ping 通主机上的网卡的网段(如果数据包没往外走的话需要确定主机是否开启了 ip_forward)
	 * 暂时没法 ping 通外网
	 * 因为此时的流量包只能往外出而不能往里进
	 * 原因是流量包往外出的时候还需要做一次 snat
	 * 没做 nat 转换的话, 外网在往回送消息的时候不知道应该往哪儿发
	 * 不过 testcni 这里暂时没有做 snat 的操作, 因为暂时没这个需求~
	 *
	 *
	 * 接下来要让不同节点上的 pod 互相通信了
	 * 可以尝试先手动操作
	 * 	1. 主机上添加路由规则: ip route add 10.244.x.0/24 via 192.168.98.x dev ens33, 也就是把非本机的节点的网段和其他 node 的 ip 做个映射
	 *  2. 其他每台集群中的主机也添加
	 *  3. 把每台主机上的对外网卡都用 iptables 设置为可 ip forward
	 * 以上手动操作可成功
	 */

	// return

	// 首先通过 ipam 获取到 etcd 中存放的集群中所有节点的相关网络信息
	networks, err := ipamClient.Get().AllHostNetwork()
	if err != nil {
		fmt.Println("这里的获取所有节点的网络信息失败, err: ", err.Error())
		return
	}

	// 然后获取一下本机的网卡信息
	currentNetwork, err := ipamClient.Get().HostNetwork()
	if err != nil {
		fmt.Println("获取本机网卡信息失败, err: ", err.Error())
		return
	}

	// 这里面要做的就是把其他节点上的 pods 的 cidr 和其主机的网卡 ip 作为一条路由规则创建到当前主机上
	err = nettools.SetOtherHostRouteToCurrentHost(networks, currentNetwork)
	if err != nil {
		fmt.Println("给主机添加其他节点网络信息失败, err: ", err.Error())
		return
	}

	// 都完事儿之后理论上同一台主机下的俩 netns(pod) 就能通信了
	// 如果无法通信, 有可能是 iptables 被设置了 forward drop
	// 需要用 iptables 允许网桥做转发
	// 接下来获取网卡信息, 把本机网卡插入到网桥上
	link, err := netlink.LinkByName(currentNetwork.Name)
	if err != nil {
		fmt.Println("获取本机网卡失败, err: ", err.Error())
		return
	}
	err = nettools.SetIptablesForDeviceToFarwordAccept(link.(*netlink.Device))
	if err != nil {
		fmt.Println("设置 ens33 转发规则失败")
		return
	}

	// // 接下来获取网卡信息, 把本机网卡插入到网桥上
	// link, err := netlink.LinkByName(currentNetwork.Name)
	// if err != nil {
	// 	fmt.Println("获取本机网卡失败, err: ", err.Error())
	// 	return
	// }

	// bridge, err := netlink.LinkByName(pluginConfig.Bridge)
	// if err != nil {
	// 	fmt.Println("获取网桥设备失败, err: ", err.Error())
	// 	return
	// }

	// err = nettools.SetDeviceMaster(link.(*netlink.Device), bridge.(*netlink.Bridge))
	// if err != nil {
	// 	fmt.Println("把网卡塞入网桥 gg, err: ", err.Error())
	// 	return
	// }

	fmt.Println("搞定!!!")
	_gw := net.ParseIP(gateway)

	_, _podIP, _ := net.ParseCIDR(podIP)

	result := &current.Result{
		CNIVersion: pluginConfig.CNIVersion,
		IPs: []*current.IPConfig{
			{
				// Version: "IPv4",
				Address: *_podIP,
				Gateway: _gw,
			},
		},
	}
	types.PrintResult(result, pluginConfig.CNIVersion)

	return
}
