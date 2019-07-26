#!/usr/bin/env python

#######################################
#
# Main Script To Trigger Test Execution
#
#######################################

import os
import sys
import getopt
import pdb
from time import gmtime, strftime


class Main:
    
    test_suite_list = []
    tags = []
 
    def __init__(self):
         pass


    @classmethod
    def arg_parser(cls, args):
        """Cmd line arg parser
        """
         
        try:
            opts, args = getopt.getopt(args, "h:i:t:", ["help=", "input-suite=", "tags=", ])
        except getopt.GetoptError as err:
            cls.help()
        for opt, arg in opts:
            if opt in ('-h', '--help'):
                cls.help()
            elif opt in ('-i', '--input-suite'):
                cls.test_suite_list = list(arg.split())
            elif opt in ('-t', '--tags'):
                cls.tags = list(arg.split())
            else:
                cls.help()


    @staticmethod
    def help():
        """ Help about how to start test execution
        """
        help_str = """

        Execute below Command to start test execution

        1. python Main.py -i <PATH to test suite> -t <Tags to execute selected test>
        
           or 
             
        2. python Main.py --input-suite <PATH to test suite> --tags <Tags to execute selected test>
 
        e.g.
           python Main.py -i 'pcc_login_test.robot'

        Options:
        -------
           -i / --input-suite    [Mandatory] Give test suite name to be execute

           -t / --tags           [Optional] Gives tags to execute selected test from suite
                                 By default all test cases from suite will be execute                                    


       Example:
       -------
       1. Execute single test suite with all test case
          python Main.py -i 'pcc_login_test.robot'

       2. Execute multiple test suite with all test case
          python Main.py -i 'pcc_login_test.robot pcc_regression_test.robot'
    
       3. Execute single test suite with selected test case
          python Main.py -i 'pcc_regression_test.robot' -t 'node_management'
  
        """
        print(help_str)
        sys.exit(0) 


    @staticmethod
    def get_current_time():
        return strftime("%Y-%m-%d_%H%M%S", gmtime())
    

    @classmethod
    def start_test_exec(cls, args):
        """ Trigger Test Execution and dump
            logs at appropriate location
        """
        cls.arg_parser(args)
        if not cls.test_suite_list:
            cls.help()

        for test_suite in list(cls.test_suite_list):
           if not os.path.isfile(str(test_suite)):
               print("test suite file not present at location: {0}".format(test_suite))
               print("skipping test execution......")
               continue
       
           time_str = cls.get_current_time()
           test_trigger_data = "robot -l ./logs/log_{0}.html -r ./logs/report_{0}.html -o ./logs/output_{0}.xml ".format(time_str)
           if cls.tags:
              for tag in list(cls.tags):
                  test_trigger_data += "-i {0} ".format(str(tag))
           test_trigger_data += "{0}".format(test_suite)
           print("starting test execution with data: {0}".format(test_trigger_data))
           os.system(test_trigger_data) 
          

# Entry point
if __name__ == '__main__':

    if len(sys.argv[1:]) < 2:
        Main.help()

    Main.start_test_exec(sys.argv[1:])
