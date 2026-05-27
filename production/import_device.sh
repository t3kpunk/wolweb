#! /bin/bash

sed -i 's/\r$//g' clients.csv

# while IFS=";" read -r name mac
# do
#   mac=`sed 's/\r$//' $mac`
#   printf "\n{\n    \"name\": \"%s\",\n    \"mac\": \"%17s\",\n    \"ip\": \"%s\"\n}," $name $mac "10.10.10.255:9"
# done < <(tail -n +2 clients.csv)

jq -Rsn '
  {"devices":
    [inputs
     | . / "\n"
     | (.[] | select(length > 0) | . / ";") as $input
     | {"name": $input[0], "mac": $input[1], "ip": "10.10.10.255:9"}]}
' < <(tail -n +2 clients.csv) > new_devices.json