#!/usr/bin/env bash

### CONFIG ###
## SETUP ##
OPENSSL=TRUE
TLS=TRUE
VPN=TRUE
GMKSEED=TRUE
IMGSTAND=TRUE
IMGWAIT=TRUE

## VAR ##
IP_FILE="ipfile.txt" #ПОЛОЖИТЬ РЯДОМ СО СКРИПТОМ ФАЙЛ С АЙПИШНИКАМИ И НАПИСАТЬ СЮДА ЕГО ИМЯ

OPVPN="openvpn-gost_2.4.11-8.16_armhf.deb"
OPSSL="stunnel-gost_5.60-5.11_armhf.deb"
CP="openssl-r_1.1.1o-10.13.emias_armhf.deb"
GMK="gmkseed_4.0.0-4.3_armhf.deb"
PARAMSSH="-o StrictHostKeyChecking=no -i ~/.ssh/id_rsa"

### FUNC ###
logevent() {

}



### MAIN ###
mapfile -t IP_LIST < "$IP_FILE"

for i in "${!IP_LIST[@]}"; do
        ip="${IP_LIST[$i]}"
        [[ -z "$ip" ]] && continue

        echo "==========$((i + 1)) with ip $ip============"

        scp -o StrictHostKeyChecking=no $CP $GMK $OPSSL $OPVPN root@$ip:~/
        if [ $? -ne 0 ]; then
                echo "Failed scp"
                exit 1
        fi

        mac=$(ssh -o StrictHostKeyChecking=no root@$ip "ip a show eth0 | awk '/link\/ether/ {print \$2}' | tr ':' '-'")
        echo $mac

        ssh -o StrictHostKeyChecking=no root@$ip "systemctl stop omini-panel-rv && dpkg -i openssl-r_1.1.1o-10.13.emias_armhf.deb \
        && dpkg -i gmkseed_4.0.0-4.3_armhf.deb && dpkg -i stunnel-gost_5.60-5.11_armhf.deb && dpkg -i openvpn-gost_2.4.11-8.16_armhf.deb"
        if [ $? -ne 0 ]; then
                echo "Failed install"
                exit 1
        fi

        ssh -o StrictHostKeyChecking=no root@$ip "cd /opt/tools && ./hwinfo > $mac 2>&1" 
        if [ $? -ne 0 ]; then
                echo "Failed info"
                exit 1
        fi

        scp -o StrictHostKeyChecking=no root@$ip:/opt/tools/$mac info/.
        if [ $? -ne 0 ]; then
                echo "Failed scp"
                exit 1
        fi

        ssh -o StrictHostKeyChecking=no root@$ip "rm openssl-r_1.1.1o-10.13.emias_armhf.deb \
        && rm gmkseed_4.0.0-4.3_armhf.deb && rm stunnel-gost_5.60-5.11_armhf.deb && rm openvpn-gost_2.4.11-8.16_armhf.deb && rm /opt/tools/$mac && systemctl start omini-panel-rv" 

        echo "succes"
done
