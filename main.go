package main

import (
	"encoding/json"
	"github.com/containernetworking/cni/pkg/version"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	"testcni/cni"
	"testcni/skel"
	"testcni/utils"
)

func cmdAdd(args *skel.CmdArgs) error {
	utils.WriteLog("Enter cmdAdd")
	utils.WriteLog(
		"CmdArgs: ", "ContainerID: ", args.ContainerID,
		"Netns: ", args.Netns,
		"IfName: ", args.IfName,
		"Args: ", args.Args,
		"Path: ", args.Path,
		"StdinData: ", string(args.StdinData))

	pluginConfig := &cni.PluginConf{}
	if err := json.Unmarshal(args.StdinData, pluginConfig); err != nil {
		utils.WriteLog("args.StdinData 转 pluginConfig 失败")
		return err
	}
	return nil
}

func cmdDel(args *skel.CmdArgs) error {
	utils.WriteLog("Enter cmdDel")
	utils.WriteLog(
		"CmdArgs: ", "ContainerID: ", args.ContainerID,
		"Netns: ", args.Netns,
		"IfName: ", args.IfName,
		"Args: ", args.Args,
		"Path: ", args.Path,
		"StdinData: ", string(args.StdinData))
	// 这里的 del 如果返回 error 的话, kubelet 就会尝试一直不停地执行 StopPodSandbox
	// 直到删除后的 error 返回 nil 未知
	// return errors.New("test cmdDel")
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	utils.WriteLog("Enter cmdCheck")
	utils.WriteLog(
		"CmdArgs: ", "ContainerID: ", args.ContainerID,
		"Netns: ", args.Netns,
		"IfName: ", args.IfName,
		"Args: ", args.Args,
		"Path: ", args.Path,
		"StdinData: ", string(args.StdinData))
	return nil
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("testcni"))
}
