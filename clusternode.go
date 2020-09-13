package main

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"os"

	"github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
)

const (
	UnusedHashSlot = iota
	NewHashSlot
	AssignedHashSlot
)

// detail info for redis node.
type NodeInfo struct {
	host       string
	port       uint

	name       string
	addr       string
	password   string
	flags      []string
	replicate  string
	pingSent   int
	pingRecv   int
	weight     int
	balance    int
	linkStatus string
	slots      map[int]int
	migrating  map[int]string
	importing  map[int]string
}

func (ni *NodeInfo) HasFlag(flag string) bool {
	for _, f := range ni.flags {
		if strings.Contains(f, flag) {
			return true
		}
	}
	return false
}

func (ni *NodeInfo) String() string {
	return fmt.Sprintf("%s:%d", ni.host, ni.port)
}

//////////////////////////////////////////////////////////
// struct of redis cluster node.
type ClusterNode struct {
	r             redis.Conn
	info          *NodeInfo
	dirty         bool
	friends       []*NodeInfo
	replicasNodes []*ClusterNode
	verbose       bool
}

func NewClusterNode(addr string) (node *ClusterNode) {

	var host, port string
	var err error

	hostport := strings.Split(addr, "@")[0]
	parts := strings.Split(hostport, ":")
	if len(parts) < 2 {
		logrus.Fatalf("Invalid IP or Port (given as %s) - use IP:Port format", addr)
		return nil
	}

	if len(parts) > 2 {
		// ipv6 in golang must like: "[fe80::1%lo0]:53", see detail in net/dial.go
		host, port, err = net.SplitHostPort(hostport)
		if err != nil {
			logrus.Fatalf("New cluster node error: %s!", err)
		}
	} else {
		host = parts[0]
		port = parts[1]
	}

	p, _ := strconv.ParseUint(port, 10, 0)

	node = &ClusterNode{
		r: nil,
		info: &NodeInfo{
			host:      host,
			port:      uint(p),
			password:  RedisPassword,
			slots:     make(map[int]int),
			migrating: make(map[int]string),
			importing: make(map[int]string),
			replicate: "",
		},
		dirty:   false,
		verbose: false,
	}

	if os.Getenv("ENV_MODE_VERBOSE") != "" {
		node.verbose = true
	}

	return node
}

func (cn *ClusterNode) Host() string {
	return cn.info.host
}

func (cn *ClusterNode) Port() uint {
	return cn.info.port
}

func (cn *ClusterNode) Name() string {
	return cn.info.name
}

func (cn *ClusterNode) HasFlag(flag string) bool {
	for _, f := range cn.info.flags {
		if strings.Contains(f, flag) {
			return true
		}
	}
	return false
}

func (cn *ClusterNode) Replicate() string {
	return cn.info.replicate
}

func (cn *ClusterNode) SetReplicate(nodeId string) {
	cn.info.replicate = nodeId
	cn.dirty = true
}

func (cn *ClusterNode) Weight() int {
	return cn.info.weight
}

func (cn *ClusterNode) SetWeight(w int) {
	cn.info.weight = w
}

func (cn *ClusterNode) Balance() int {
	return cn.info.balance
}

func (cn *ClusterNode) SetBalance(balance int) {
	cn.info.balance = balance
}

func (cn *ClusterNode) Slots() map[int]int {
	return cn.info.slots
}

func (cn *ClusterNode) Migrating() map[int]string {
	return cn.info.migrating
}

func (cn *ClusterNode) Importing() map[int]string {
	return cn.info.importing
}

func (cn *ClusterNode) R() redis.Conn {
	return cn.r
}

func (cn *ClusterNode) Info() *NodeInfo {
	return cn.info
}

func (cn *ClusterNode) IsDirty() bool {
	return cn.dirty
}

func (cn *ClusterNode) Friends() []*NodeInfo {
	return cn.friends
}

func (cn *ClusterNode) ReplicasNodes() []*ClusterNode {
	return cn.replicasNodes
}

func (cn *ClusterNode) AddReplicasNode(node *ClusterNode) {
	cn.replicasNodes = append(cn.replicasNodes, node)
}

func (cn *ClusterNode) String() string {
	return cn.info.String()
}

func (cn *ClusterNode) NodeString() string {
	return cn.info.String()
}

func (cn *ClusterNode) Connect(abort bool) (err error) {
	var addr string

	if cn.r != nil {
		return nil
	}

	if strings.Contains(cn.info.host, ":") {
		// ipv6 in golang must like: "[fe80::1%lo0]:53", see detail in net/dial.go
		addr = fmt.Sprintf("[%s]:%d", cn.info.host, cn.info.port)
	} else {
		addr = fmt.Sprintf("%s:%d", cn.info.host, cn.info.port)
	}
	//client, err := redis.DialTimeout("tcp", addr, 0, 1*time.Second, 1*time.Second)
	var client redis.Conn
	if cn.info.password != "" {
		client, err = redis.Dial("tcp", addr, redis.DialConnectTimeout(60*time.Second), redis.DialPassword(cn.info.password))
	} else {
		client, err = redis.Dial("tcp", addr, redis.DialConnectTimeout(60*time.Second))
	}
	if err != nil {
		if abort {
			logrus.Fatalf("Sorry, connect to node %s failed in abort mode!", addr)
		} else {
			logrus.Errorf("Sorry, can't connect to node %s!", addr)
			return err
		}
	}

	if _, err = client.Do("PING"); err != nil {
		if abort {
			logrus.Fatalf("Sorry, ping node %s failed in abort mode!", addr)
		} else {
			logrus.Errorf("Sorry, ping node %s failed!", addr)
			return err
		}
	}

	if cn.verbose {
		logrus.Printf("Connecting to node %s OK", cn.String())
	}

	cn.r = client
	return nil
}

func (cn *ClusterNode) Call(cmd string, args ...interface{}) (interface{}, error) {
	err := cn.Connect(true)
	if err != nil {
		return nil, err
	}

	return cn.r.Do(cmd, args...)
}

func (cn *ClusterNode) Dbsize() (int, error) {
	return redis.Int(cn.Call("DBSIZE"))
}

func (cn *ClusterNode) ClusterAddNode(addr string) (ret string, err error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil || host == "" || port == "" {
		return "", fmt.Errorf("Bad format of host:port: %s!", addr)
	}
	return redis.String(cn.Call("CLUSTER", "meet", host, port))
}

func (cn *ClusterNode) ClusterReplicateWithNodeID(nodeid string) (ret string, err error) {
	return redis.String(cn.Call("CLUSTER", "replicate", nodeid))
}

func (cn *ClusterNode) ClusterForgetNodeID(nodeid string) (ret string, err error) {
	return redis.String(cn.Call("CLUSTER", "forget", nodeid))
}

func (cn *ClusterNode) ClusterNodeShutdown() (err error) {
	cn.r.Send("SHUTDOWN")
	if err = cn.r.Flush(); err != nil {
		return err
	}
	return nil
}

func (cn *ClusterNode) ClusterCountKeysInSlot(slot int) (int, error) {
	return redis.Int(cn.Call("CLUSTER", "countkeysinslot", slot))
}

func (cn *ClusterNode) ClusterGetKeysInSlot(slot int, pipeline int) (string, error) {
	return redis.String(cn.Call("CLUSTER", "getkeysinslot", slot, pipeline))
}

func (cn *ClusterNode) ClusterSetSlot(slot int, cmd string) (string, error) {
	return redis.String(cn.Call("CLUSTER", "setslot", slot, cmd, cn.Name()))
}

func (cn *ClusterNode) AssertCluster() bool {
	info, err := redis.String(cn.Call("INFO", "cluster"))
	if err != nil ||
		!strings.Contains(info, "cluster_enabled:1") {
		return false
	}

	return true
}

func (cn *ClusterNode) AssertEmpty() bool {

	info, err := redis.String(cn.Call("CLUSTER", "INFO"))
	db0, e := redis.String(cn.Call("INFO", "db0"))
	if err != nil || !strings.Contains(info, "cluster_known_nodes:1") ||
		e != nil || strings.Trim(db0, " ") != "" {
		logrus.Fatalf("Node %s is not empty. Either the node already knows other nodes (check with CLUSTER NODES) or contains some key in database 0.", cn.String())
	}

	return true
}

func (cn *ClusterNode) LoadInfo(getfriends bool) (err error) {
	var result string
	if result, err = redis.String(cn.Call("CLUSTER", "NODES")); err != nil {
		return err
	}

	nodes := strings.Split(result, "\n")
	for _, val := range nodes {
		// name addr flags role ping_sent ping_recv link_status slots
		parts := strings.Split(val, " ")
		if len(parts) <= 3 {
			continue
		}

		sent, _ := strconv.ParseInt(parts[4], 0, 32)
		recv, _ := strconv.ParseInt(parts[5], 0, 32)
		addr := strings.Split(parts[1], "@")[0]
		host, port, _ := net.SplitHostPort(addr)
		p, _ := strconv.ParseUint(port, 10, 0)

		node := &NodeInfo{
			name:       parts[0],
			addr:       parts[1],
			flags:      strings.Split(parts[2], ","),
			replicate:  parts[3],
			pingSent:   int(sent),
			pingRecv:   int(recv),
			linkStatus: parts[6],

			host:      host,
			port:      uint(p),
			slots:     make(map[int]int),
			migrating: make(map[int]string),
			importing: make(map[int]string),
		}

		if parts[3] == "-" {
			node.replicate = ""
		}

		if strings.Contains(parts[2], "myself") {
			if cn.info != nil {
				cn.info.name = node.name
				cn.info.addr = node.addr
				cn.info.flags = node.flags
				cn.info.replicate = node.replicate
				cn.info.pingSent = node.pingSent
				cn.info.pingRecv = node.pingRecv
				cn.info.linkStatus = node.linkStatus
			} else {
				cn.info = node
			}

			for i := 8; i < len(parts); i++ {
				slots := parts[i]
				if strings.Contains(slots, "<") {
					slotStr := strings.Split(slots, "-<-")
					slotId, _ := strconv.Atoi(slotStr[0])
					cn.info.importing[slotId] = slotStr[1]
				} else if strings.Contains(slots, ">") {
					slotStr := strings.Split(slots, "->-")
					slotId, _ := strconv.Atoi(slotStr[0])
					cn.info.migrating[slotId] = slotStr[1]
				} else if strings.Contains(slots, "-") {
					slotStr := strings.Split(slots, "-")
					firstId, _ := strconv.Atoi(slotStr[0])
					lastId, _ := strconv.Atoi(slotStr[1])
					cn.AddSlots(firstId, lastId)
				} else {
					firstId, _ := strconv.Atoi(slots)
					cn.AddSlots(firstId, firstId)
				}
			}
		} else if getfriends {
			cn.friends = append(cn.friends, node)
		}
	}
	return nil
}

func (cn *ClusterNode) AddSlots(start, end int) {
	for i := start; i <= end; i++ {
		cn.info.slots[i] = NewHashSlot
	}
	cn.dirty = true
}

func (cn *ClusterNode) FlushNodeConfig() {
	if !cn.dirty {
		return
	}

	if cn.Replicate() != "" {
		// run replicate cmd
		if _, err := cn.ClusterReplicateWithNodeID(cn.Replicate()); err != nil {
			// If the cluster did not already joined it is possible that
			// the slave does not know the master node yet. So on errors
			// we return ASAP leaving the dirty flag set, to flush the
			// config later.
			return
		}
	} else {
		var array []int
		for slot, value := range cn.Slots() {
			if value == NewHashSlot {
				array = append(array, slot)
				cn.info.slots[slot] = AssignedHashSlot
				_, err := cn.ClusterAddSlots(slot)
				if err != nil {
					logrus.Printf("ClusterAddSlots slot: %d with error %s", slot, err)
					return
				}
			}
		}

	}
	cn.dirty = false
}

// XXX: check the error for call CLUSTER addslots
func (cn *ClusterNode) ClusterAddSlots(args ...interface{}) (ret string, err error) {
	return redis.String(cn.Call("CLUSTER", "addslots", args[0]))
}

// XXX: check the error for call CLUSTER delslots
func (cn *ClusterNode) ClusterDelSlots(args ...interface{}) (ret string, err error) {
	return redis.String(cn.Call("CLUSTER", "delslots", args))
}

func (cn *ClusterNode) ClusterBumpepoch() (ret string, err error) {
	return redis.String(cn.Call("CLUSTER", "bumpepoch"))
}

func (cn *ClusterNode) InfoString() (result string) {
	var role = "M"

	if !cn.HasFlag("master") {
		role = "S"
	}

	keys := make([]int, 0, len(cn.Slots()))

	for k := range cn.Slots() {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	slotstr := MergeNumArray2NumRange(keys)

	if cn.Replicate() != "" && cn.dirty {
		result = fmt.Sprintf("S: %s %s", cn.info.name, cn.String())
	} else {
		// fix myself flag not the first element of []slots
		result = fmt.Sprintf("%s: %s %s\n\t   slots:%s (%d slots) %s",
			role, cn.info.name, cn.String(), slotstr, len(cn.Slots()), strings.Join(cn.info.flags[1:], ","))
	}

	if cn.Replicate() != "" {
		result = result + fmt.Sprintf("\n\t   replicates %s", cn.Replicate())
	} else {
		result = result + fmt.Sprintf("\n\t   %d additional replica(s)", len(cn.replicasNodes))
	}

	return result
}

func (cn *ClusterNode) GetConfigSignature() string {
	config := []string{}

	result, err := redis.String(cn.Call("CLUSTER", "NODES"))
	if err != nil {
		return ""
	}

	nodes := strings.Split(result, "\n")
	for _, val := range nodes {
		parts := strings.Split(val, " ")
		if len(parts) <= 3 {
			continue
		}

		sig := parts[0] + ":"

		slots := []string{}
		if len(parts) > 7 {
			for i := 8; i < len(parts); i++ {
				p := parts[i]
				if !strings.Contains(p, "[") {
					slots = append(slots, p)
				}
			}
		}
		if len(slots) == 0 {
			continue
		}
		sort.Strings(slots)
		sig = sig + strings.Join(slots, ",")

		config = append(config, sig)
	}

	sort.Strings(config)
	return strings.Join(config, "|")
}

///////////////////////////////////////////////////////////
// some useful struct contains cluster node.
type ClusterArray []ClusterNode

func (c ClusterArray) Len() int {
	return len(c)
}

func (c ClusterArray) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c ClusterArray) Less(i, j int) bool {
	return len(c[i].Slots()) < len(c[j].Slots())
}

type MovedNode struct {
	Source ClusterNode
	Slot   int
}
