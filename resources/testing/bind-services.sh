set -eu
# unbind first
cf t -o it4it-org -s spaceA
cf unbind-service appA src1
cf unbind-service appB src1
cf unbind-service appC src1
cf t -o it4it-org -s spaceB
cf unbind-service appD dest1
cf unbind-service appE dest1
cf t -o it4it-org -s spaceC
cf unbind-service appF src2
cf unbind-service appG src2
cf unbind-service appH src2
cf t -o it4it-org -s spaceD
cf unbind-service appI dest2
cf unbind-service appJ dest2
cf unbind-service appJ dest3

# bind again
cf t -o it4it-org -s spaceA
cf bs appA src1
cf bs appB src1
cf bs appC src1
cf t -o it4it-org -s spaceB
cf bs appD dest1
cf bs appE dest1 -c '{"protocol":"udp"}'
cf t -o it4it-org -s spaceC
cf bs appF src2
cf bs appG src2
cf bs appH src2
cf t -o it4it-org -s spaceD
cf bs appI dest2 -c '{"port":8443,"protocol":"udp"}'
cf bs appJ dest2
cf bs appJ dest3 -c '{"port":8443}'

# unbind again
cf t -o it4it-org -s spaceA
cf unbind-service appA src1
cf unbind-service appB src1
cf unbind-service appC src1
cf t -o it4it-org -s spaceB
cf unbind-service appD dest1
cf unbind-service appE dest1
cf t -o it4it-org -s spaceC
cf unbind-service appF src2
cf unbind-service appG src2
cf unbind-service appH src2
cf t -o it4it-org -s spaceD
cf unbind-service appI dest2
cf unbind-service appJ dest2
cf unbind-service appJ dest3

# bind again in a different order
cf t -o it4it-org -s spaceB
cf bs appD dest1
cf bs appE dest1 -c '{"protocol":"udp"}'
cf t -o it4it-org -s spaceA
cf bs appA src1
cf bs appB src1
cf bs appC src1
cf t -o it4it-org -s spaceD
cf bs appI dest2 -c '{"port":8443,"protocol":"udp"}'
cf bs appJ dest2
cf bs appJ dest3 -c '{"port":8443}'
cf t -o it4it-org -s spaceC
cf bs appF src2
cf bs appG src2
cf bs appH src2
