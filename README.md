#k8s-cni-test


## IPIP mode test method
0. It is best to have a clean k8s environment without any network plug-ins installed.

1.
```js
// Create a new file ending with .conf in the /etc/cni/net.d/ directory and enter the following configuration items
{
  "cniVersion": "0.3.0",
  "name": "testcni",
  "type": "testcni",
  "mode": "ipip",
  "subnet": "10.244.0.0/16"
}
```

2. 
```bash
# Execute in the project root directory
make build_main
```

3. A main will be generated at this time

4. Go to https://github.com/projectcalico/bird clone project

5. Execution
```
# Compile calico's bird
# After compilation, there will be a bird binary in the dist directory
ARCH=<your computer architecture> ./build.sh
```

6. Create the /opt/testcni directory and copy the bird binary above to here

7. Copy the main binary generated in the third step to /opt/cni/bin/testcni
```bash
mv main /opt/cni/bin/testcni
```
</br>
</br>
</br>

---
</br>

## vxlan mode test method
0. It is best to have a clean k8s environment without any network plug-ins installed.

1.
```js
// Create a new file ending with .conf in the /etc/cni/net.d/ directory and enter the following configuration items
{
  "cniVersion": "0.3.0",
  "name": "testcni",
  "type": "testcni",
  "mode": "vxlan",
  "subnet": "10.244.0.0"
}
```

2.
```bash
# Execute in the project root directory
make build
```
3. A binary file named testcni will be generated. Three ebpf files will be generated at the same time. These three ebpf files will be automatically copied to the “/opt/testcni/” directory. If this directory does not exist, you can create it manually.

4. Copy the testcni generated in the previous step to the "/opt/cni/bin" directory
</br>
</br>
</br>

---
</br>

## IPVlan & MACVlan mode test method
0. It is best to have a clean k8s environment without any network plug-ins installed.

1.
```js
//Create a new file ending with .conf in the /etc/cni/net.d/ directory of each node and enter the following configuration items
// Note that the range in "subnet" and "ipam" needs to be manually changed to your own environment. In addition, each node of the range should be configured to a different range.
{
  "cniVersion": "0.3.0",
  "name": "testcni",
  "type": "testcni",
  "mode": "ipvlan",
  "subnet": "192.168.64.0/24",
  "ipam": {
    "rangeStart": "192.168.64.90",
    "rangeEnd": "192.168.64.100"
  }
}

2. 
```bash
# Execute in the project root directory
make build_main
```

3. At this time, a main binary will be generated, and the binary will be copied to /opt/cni/bin/testcni

```bash
mv main /opt/cni/bin/testcni
```
</br>
</br>
</br>

---
</br>

## host-gw mode test method
1. 
```js
// Create a new file ending with .conf in the /etc/cni/net.d/ directory and enter the following configuration items
{
  "cniVersion": "0.3.0",
  "name": "testcni",
  "type": "testcni",
  "bridge": "testcni0",
  "subnet": "10.244.0.0/16"
}
```
2. Change the IP address used to initialize the etcd client under /etcd/client.go to the etcd address of your own cluster.
3. go build main.go
4. mv main /opt/cni/bin/testcni
5. Repeat the above three steps on each host.
6. kubectl apply -f test-busybox.yaml
7. Check cluster pod status

</br></br>

## Test without k8s cluster
1. Testing can be done through main_test.go in the /test directory
2. Before testing, create a command space using ip netns add test.net.1
3. Then go test ./test/main_test.go -v
4. Then perform the same steps on other nodes.
5. ip netns exec test.net.1 ping the network card ip under ns on another node

</br></br>

## Use the cnitool test provided by the k8s cni repo
1. Switch to the test/cni-test branch
2. Enter the ./cnitool directory
3. go build cnitool.go
4. ip netns add test.net.1 creates a net ns
5. Create the same configuration as above in the /etc/cni/net.d/ directory
6. ./cnitool add testcni /run/netns/test.net.1

</br></br>

## Problems and troubleshooting
### Troubleshooting methods when encountering problems
1. View kubelet logs through journalctl -xeu kubelet -f command
2. Modify the log output address in the ./utils/write_log.go file. Key error messages will be automatically sent to this address.

### Problems you may encounter:
1. If the compiled main.go has been copied to /opt/cni/bin/testcni but kubelet still reports an error like "not found", try adding "export CNI_PATH=/ to the environment variable opt/cni/bin"
2. If the kubelet log shows something like "the configuration file contains illegal characters", check whether any logs are output directly to the standard output using fmt in all codes. cni reads the configuration through the standard output, so if there is any illegal Configuration related information is output, it will definitely be gg

</br></br>
