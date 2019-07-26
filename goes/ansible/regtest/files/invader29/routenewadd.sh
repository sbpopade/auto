#!/bin/bash
for ((i=1; i<=6; i++))
        do
                for ((j=1; j<=255; j++))

                        do
                        var="1.1.${i}.${j}"
                        route add -net $var netmask 255.255.255.255 gw 10.0.5.31
                        done
        done
