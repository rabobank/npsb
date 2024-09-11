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
cf cs network-policies default src1 -c '{"type":"source","name":"source1","description":"doe_iets_leuks-src1","scope":"global"}'
cf t -o it4it-org -s spaceB
cf cs network-policies default dest1 -c '{"type":"destination","source":"source1"}'
cf t -o it4it-org -s spaceC
cf cs network-policies default src2 -c '{"type":"source","name":"source2","description":"doe_iets_leuks-src2","scope":"global"}'
cf t -o it4it-org -s spaceD
cf cs network-policies default dest2 -c '{"type":"destination","source":"source1"}'
cf cs network-policies default dest3 -c '{"type":"destination","source":"source2"}'
cf t -o it4it-org -s spaceE
cf cs network-policies default src3 -c '{"type":"source","name":"source3","description":"doe_iets_leuks-src3","scope":"local"}'
cf cs network-policies default dest4 -c '{"type":"destination","source":"source3"}'
cf cs network-policies default src4 -c '{"type":"source","name":"source4","description":"doe_iets_leuks-src4","scope":"local"}'
cf cs network-policies default dest5 -c '{"type":"destination","source":"source4"}'

# show service instances:
cf curl "/v3/service_instances?label_selector=rabobank.com/npsb.type"|jq -r '.resources[]|.guid , .name, .metadata.labels'
