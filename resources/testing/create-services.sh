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

cf t -o it4it-org -s spaceA
cf cs network-policies default src1 -c '{"type":"source","name":"source1","description":"doe iets leuks src1"}'
cf t -o it4it-org -s spaceB
cf cs network-policies default dest1 -c '{"type":"destination","sourceName":"source1","sourceSpace":"spaceA","sourceOrg":"it4it-org"}'
cf t -o it4it-org -s spaceC
cf cs network-policies default src2 -c '{"type":"source","name":"source2","description":"doe iets leuks src2"}'
cf t -o it4it-org -s spaceD
cf cs network-policies default dest2 -c '{"type":"destination","sourceName":"source1","sourceSpace":"spaceA","sourceOrg":"it4it-org"}'
cf cs network-policies default dest3 -c '{"type":"destination","sourceName":"source2","sourceSpace":"spaceC","sourceOrg":"it4it-org"}'

# show service instances:
cf curl "/v3/service_instances?label_selector=rabobank.com/npsb.type"|jq -r '.resources[]|.guid , .name, .metadata.labels'
