#!/usr/bin/python3
import argparse
import datetime
import os
import sys
import time
from collections import namedtuple
from subprocess import call
from subprocess import check_output

Tech = namedtuple('Tech', 'name cmdlist')

# run each cmdlist, add timestamp to name
# these commands run at start and end of show tech to get incremental stats
TechTimeStampList = [
    Tech(name="vnet_stats",
         cmdlist={
             "goes vnet show errors",
             "goes vnet show hardware",
         }),
    Tech(name="fe1_stats",
         cmdlist={
             "goes vnet show fe1 int",
             "goes vnet show fe1 pipe",
         }),
]

# run each cmdlist, write output to 'name'
TechList = [
    Tech(name="hardware",
         cmdlist={
             "goes hget platina-mk1 eeprom",
             "goes hget platina-mk1 qsfp",
             "goes vnet show fe1 switches",
             "goes vnet show fe1 temp",
         }),
    Tech(name="system",
         cmdlist={
             "date",
             "uname -a",
             "uptime",
             "cat /etc/modprobe.d/goesd-platina-mk1-modprobe.conf",
             "modinfo platina-mk1",
             "ethtool -i xeth1",
         }),
    Tech(name="journalctl",
         cmdlist={
             "journalctl",
         }),
    Tech(name="ip",
         cmdlist={
             "arp -v",
             "ip route",
             "ip neighbor",
             "ip netns",
             "ip -s link",
         }),
    Tech(name="interfaces",
         cmdlist={
             "cat /etc/network/interfaces",
             "grep -H . /etc/network/interfaces.d/*",
         }),
    Tech(name="xeth_util",
         cmdlist={
             "./xeth_util.sh netns_showup",
             "./xeth_util.sh netns_show ip route vrf",
         }),
    Tech(name="docker",
         cmdlist={
             "docker ps",
         }),
    Tech(name="lldp",
         cmdlist={
             "cat /etc/lldpd.conf",
             "lldpcli show config",
             "lldpcli show neighbor",
         }),
    Tech(name="goes",
         cmdlist={
             "goes status",
             "goes version",
         }),
    Tech(name="phy",
         cmdlist={
             "goes vnet show fe1 port phy",
             "goes vnet show fe1 serdes",
         }),
    Tech(name="fe1",
         cmdlist={
             "goes vnet show fe1 port-map vlan",
             "goes vnet show fe1 port-tab",
             "goes vnet show fe1 vlan",
             "goes vnet show fe1 acl l2",
             "goes vnet show fe1 acl l3",
             "goes vnet show fe1 l2",
             "goes vnet show fe1 station",
             "goes vnet show fe1 vlan br",
             "goes vnet show fe1 station",
             "goes vnet show fe1 vlan rx",
             "goes vnet show fe1 vlan tx",
             "goes vnet show fe1 tcam",
             "goes vnet show fe1 l3 rx",
             "goes vnet show fe1 l3 tx",
         }),
    Tech(name="vnet",
         cmdlist={
             "goes vnet show ports",
             "goes vnet show int",
             "goes vnet show buf",
             "goes vnet show ip fib",
             "goes vnet show neighbor",
             "goes vnet show br",
         }),
]


# ISO standard date, used in filename
# 2019-01-16T21:31:06
def techTime():
    ts = time.time()
    st = datetime.datetime.fromtimestamp(ts).strftime('%Y%m%d_%H%M%S_%f')[:-3]
    return st

def dumpTime(path):
    for d in TechTimeStampList:
        print(d.name)
        with open(path + "/" + d.name + "_" + techTime() + ".log", "w") as f:
            for c in d.cmdlist:
                f.write("# show_tech " + techTime() + " '" + c + "'\n")
                f.flush()
                call(c, stdout=f, stderr=f, shell=True)

def dump(path):

    dumpTime(path)
    
    for d in TechList:
        print(d.name)
        with open(path + "/" + d.name + ".log", "w") as f:
            for c in d.cmdlist:
                f.write("# show_tech " + techTime() + " '" + c + "'\n")
                f.flush()
                call(c, stdout=f, stderr=f, shell=True)

    dumpTime(path)


parser = argparse.ArgumentParser()
parser.add_argument('-path', '--path', help='directory')
parser.add_argument('-hash_name', '--hash_name', help='test_case name with time stamp')

args = parser.parse_args()
if args.path == None:
    print("--path is required")
    exit(1)

if args.hash_name == None:
    print("--hash_name is required")
    exit(1)

logTime = techTime()
showDir = args.path + "/" + logTime
os.makedirs(showDir)

dump(showDir)
tarfile = "../" + args.hash_name + ".tar"
status = check_output("cd " + showDir + "; tar cfz " + tarfile + " .", shell=True)