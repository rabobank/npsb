set -eu
cf t -o it4it-org -s spaceA
for A in appA appB appC
do
  cf delete -f -r "$A"
done
cf t -o it4it-org -s spaceB
for A in appD appE
do
  cf delete -f -r "$A"
done
cf t -o it4it-org -s spaceC
for A in appF appG appH
do
  cf delete -f -r "$A"
done
cf t -o it4it-org -s spaceD
for A in appI appJ
do
  cf delete -f -r "$A"
done
cf t -o it4it-org -s spaceE
for A in appL appM appN appO
do
  cf delete -f -r "$A"
done
