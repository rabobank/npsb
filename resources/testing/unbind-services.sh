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
