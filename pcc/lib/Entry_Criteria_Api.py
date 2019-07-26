#!/usr/bin/env python

################################################
#
# Entry Criteria data parsing and validation API
#
################################################

import os
import subprocess
import time
from robot.api import logger
from robot.libraries.OperatingSystem import OperatingSystem
from SSHLibrary import SSHLibrary


def robot_logger(msg):
    """Custom logger
    """
    msg_str = "\n{0}\n".format(msg)
    logger.info(str(msg_str), html=True, also_console = True)


class Entry_Criteria_Api(OperatingSystem, SSHLibrary):

    def __init__(self):
        self.master_node = None

    def get_available_node_data(self, resp_data):
        """Check any Invader or Server is available in node list
           if yes then return node with attached IP
        """
        node_data = {}
        node_id_lst = []

        try:
            if not resp_data['Data']:
                return False, None, None
            else:
                for index, data in enumerate(eval(str(resp_data))['Data']):
                    node_data.update({index:{"name":data['Name'], "ID" : str(data["Id"]), \
                                             "HOST" : str(data["Host"])}})
                    node_id_lst.append(int(data['Id']))
                robot_logger("Available Node Data = {0}".format(str(node_data)))
                return True, node_id_lst, node_data
            return False, None, None
        except Exception:
            return False, None, None

    def get_node_type(self, node_data):
        """ Get Node Type(Server Or Invader)
        """
        node_type_dict = {}

        try:
            for data in node_data:
                cmd = "ssh-keygen -f \"/home/pcc/.ssh/known_hosts\" -R {0}".format(node_data[data]["HOST"])
                try:
                    code, output = self.run_and_return_rc_and_output(cmd)
                except:
                    pass
                robot_logger("cmd-{0} o/p={1}".format(str(cmd), output))

                cmd = "ssh {0} \'{1}\'".format(node_data[data]["HOST"], "goes status")
                try:
                    code, output = self.run_and_return_rc_and_output(cmd)
                except:
                    pass
                robot_logger("cmd-{0} o/p={1}".format(str(cmd), output))

                if "goes: command not found" in output:
                    robot_logger("Node {0} is Server".format(node_data[data]["name"]))
                    node_type_dict.update({node_data[data]["HOST"] : "Server"})
                elif "No route to host" in output:
                    robot_logger("Node {0} is not connected with PCC".format(node_data[data]["name"]))
                else:
                    robot_logger("Node {0} is an Invader".format(node_data[data]["name"]))
                    node_type_dict.update({node_data[data]["HOST"] : "Invader"})
            return node_type_dict
        except:
            return node_type_dict

    def clean_invader(self, ip_addr):
        """ Clean Invader from Backend
        """
        try:
            login_op = self.open_connection(ip_addr)
            robot_logger("login over invader Ip = ", ip_addr)
            self.login("pcc", "cals0ft")

            out = self.execute_command("sudo -s")
            robot_logger("cmd = {} and o/p = {1}".format("sudo -s", out))

            out = self.execute_command("crontab -r")
            robot_logger("cmd = {} and o/p = {1}".format("crontab -r", out))
            time.sleep(1)

            out = self.execute_command("ps -aef | grep supervisord | awk '{print $2}' | xargs kill -9")
            robot_logger("cmd = {} and o/p = {1}".format("kill supervisord", out))
            time.sleep(1)

            out = self.execute_command("ps -aef | grep tinyproxy | awk '{print $2}' | xargs kill -9")
            robot_logger("cmd = {} and o/p = {1}".format("kill tinyproxy", out))
            time.sleep(1)
                       
            out = self.execute_command("ps -aef | grep dnsmasq | awk '{print $2}' | xargs kill -9")
            robot_logger("cmd = {} and o/p = {1}".format("kill dnsmasq", out))                  
            time.sleep(1)

            out = self.execute_command("ps -aef | grep dnsmasq | awk '{print $2}' | xargs kill -9")
            robot_logger("cmd = {} and o/p = {1}".format("kill dnsmasq", out))
            time.sleep(1)
                     
            out = self.execute_command("ps -aef | grep pccagent | awk '{print $2}' | xargs kill -9")
            robot_logger("cmd = {} and o/p = {1}".format("kill pccagent ", out))
            time.sleep(1)

            out = self.execute_command("ps -aef | grep collector | awk '{print $2}' | xargs kill -9")
            robot_logger("cmd = {} and o/p = {1}".format("kill collector ", out))
            time.sleep(1)

            out = self.execute_command("service maas-ROOT stop")
            robot_logger("cmd = {} and o/p = {1}".format("service maas-ROOT stop", out))
            time.sleep(3)

            out = self.execute_command("apt-get remove -y dnsmasq tinyproxy supervisor lighttpd")
            robot_logger("cmd = {} and o/p = {1}".format("apt-get remove -y dnsmasq tinyproxy supervisor lighttpd", out))
            time.sleep(10)
                      
            out = self.execute_command("rm -rf /srv/maas/")
            robot_logger("cmd = {} and o/p = {1}".format("rm -rf /srv/maas/", out))
            time.sleep(1)

            out = self.execute_command("rm -rf /etc/network/interfaces.d/maas-xeth*")
            robot_logger("cmd = {} and o/p = {1}".format("rm -rf /etc/network/interfaces.d/maas-xeth*", out))
            time.sleep(1)
            # Terminating the session with Invader
            self.close_connection()
        except:
              pass

    def clear_server(self, ip_addr):
        """ Clear server From Backend
        """
        cmd = "ssh-keygen -f \"/home/pcc/.ssh/known_hosts\" -R {0}".format(ip_addr)
        try:
            code, output = self.run_and_return_rc_and_output(cmd)
        except:
            pass
        robot_logger("cmd-{0} o/p={1}".format(str(cmd), output))

        cmd = "ssh {0} \'{1}\'".format((ip_addr, "sudo crontab -r"))
        try:
            code, output = self.run_and_return_rc_and_output(cmd)
        except:
            pass
        robot_logger("cmd-{0} o/p={1}".format(str(cmd), output))

        cmd = "ssh {0} \'{1}\'".format((ip_addr, "sudo ps -aef | grep pccagent | awk '{print $2}' | xargs kill -9"))
        try:
            code, output = self.run_and_return_rc_and_output(cmd)
        except:
            pass
        robot_logger("cmd-{0} o/p={1}".format(str(cmd), output))

        cmd = "ssh {0} \'{1}\'".format((ip_addr, "sudo ps -aef | grep collector | awk '{print $2}' | xargs kill -9"))
        try:
            code, output = self.run_and_return_rc_and_output(cmd)
        except:
            pass
        robot_logger("cmd-{0} o/p={1}".format(str(cmd), output))

    def node_clean_up_from_back_end_command(self, node_type):
        """ Clean Node Server or Invader from back End
        """
        try:
            for ip, type in eval(str(node_type)).items():
                if str(type) == "Server":
                    self.clean_invader(ip)
                elif str(type) == "Invader":
                    self.clean_server(ip)
            return True
        except:
            return False

    def server_pxe_boot(self, server_ip):
        """ PXE boot to Server
        """
        try:
            cmd_1 = "ipmitool -I lanplus -H {0} -U ADMIN -P ADMIN chassis bootdev pxe".format(server_ip)
            try:
                code, output = self.run_and_return_rc_and_output(cmd_1)
            except:
                pass
            robot_logger("cmd-{0} o/p={1}".format(str(cmd_1), output))
            
            time.sleep(5)
 
            cmd_2 = "ipmitool -I lanplus -H {0} -U ADMIN -P ADMIN chassis power cycle".format(server_ip)
            try:
                code, output = self.run_and_return_rc_and_output(cmd_2)
            except:
                pass
            robot_logger("cmd-{0} o/p={1}".format(str(cmd_2), output))
            return True
        except:
            return False
    
    def get_server_id(self, resp_data, bmc_host):
        """Get Server ID added After PXE boot
        """
        try:
            if resp_data['Data'] != None:
                for data in eval(str(resp_data))['Data']:
                    if data.has_key('bmc'):
                    	if str(data['bmc']) == str(bmc_host):
                            return True, str(data['Id'])
                return False, None
            else:
                return False, None
        except Exception:
            return False, None

    def verify_kubernetes_cluster_installed(self, ip_addr):
        """Verify Cluster Created
        """
        login_op = self.open_connection(ip_addr)
        robot_logger("login over invader Ip = ", ip_addr)
        self.login("pcc", "cals0ft")

        out = self.execute_command("sudo -s")
        robot_logger("cmd = {} and o/p = {1}".format("crontab -r", out))
        time.sleep(1)

        out = self.execute_command("kubectl get nodes")
        robot_logger("cmd = {} and o/p = {1}".format("kubectl get nodes", out))
        time.sleep(1)

        if "Ready    master" in out:
            self.master_node = str(out.split('\n')[1].split()[0])
            self.close_connection()
            return True
        else:
            self.close_connection()
            return False

    def get_k8s_installation_status(self, stat1, stat2):
        """Verify master Node
        """
        if (stat1 == True) or (stat2 == True):
            return True
        else:
            return False
