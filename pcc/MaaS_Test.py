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

    test_suite_list = None
    run_count = 1
 
    def __init__(self):
         pass


    @classmethod
    def arg_parser(cls, args):
        """Cmd line arg parser
        """

        try:
            opts, args = getopt.getopt(args, "h:i:c:", ["help=", "input-suite=", "run_cnt=", ])
        except getopt.GetoptError as err:
            cls.help()
        for opt, arg in opts:
            if opt in ('-h', '--help'):
                cls.help()
            elif opt in ('-i', '--input-suite'):
                cls.test_suite_list = str(arg)
            elif opt in ('-c', '--run_cnt'):
                try:
                    cls.run_count = int(arg)
                except:
                    print("\n\nnumber expected as run count....") 
                    cls.help()
            else:
                cls.help()


    @staticmethod
    def help():
        """ Help about how to start test execution
        """
        help_str = """

        Execute below Command to start test execution

        1. python Main.py -i <PATH to test suite> -c <Num of Time Repeat test>
        
           or 
 
        2. python Main.py --input-suite <PATH to test suite> --"run_cnt <Num of Time Repeat test>
 
        e.g.
           python Main.py -i 'pcc_login_test.robot' -c 2 

        Options:
        -------
           -i / --input-suite    [Mandatory] Give test suite name to be execute

           -c / --run-cnt         [Optional] Gives tags to execute selected test from suite
                                  By default all test cases from suite will be execute                                    


       Example:
       -------
       1. Execute single test suite with all test case
          python Main.py -i 'pcc_login_test.robot'

  
       2. Execute single test number of times
          python Main.py -i 'pcc_login_test.robot' -c 5

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
        pdb.set_trace()
        for count in range(cls.run_count):
           if not os.path.isfile(str(cls.test_suite_list)):
               print("test suite file not present at location: {0}".format(cls.test_suite_list))
               print("skipping test execution......")
               continue
       
           time_str = cls.get_current_time()
           test_trigger_data = "robot -l ./logs/log_{0}.html -r ./logs/report_{0}.html -o ./logs/output_{0}.xml ".format(time_str)
           
           test_trigger_data += "{0}".format(cls.test_suite_list)
           print("starting test execution with data: {0}".format(test_trigger_data))
           os.system(test_trigger_data) 
 

# Entry point
if __name__ == '__main__':

    if len(sys.argv[1:]) < 2:
        Main.help()

    Main.start_test_exec(sys.argv[1:])
