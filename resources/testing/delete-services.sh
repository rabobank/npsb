set -eu

# delete first
cf t -o it4it-org -s spaceA
cf ds -f src1
cf t -o it4it-org -s spaceB
cf ds -f dest1
cf t -o it4it-org -s spaceC
cf ds -f src2
cf t -o it4it-org -s spaceD
cf ds -f dest2
cf ds -f dest3

# show service instances:
cf curl "/v3/service_instances?label_selector=rabobank.com/npsb.type"|jq -r '.resources[]|.guid , .name, .metadata.labels'
