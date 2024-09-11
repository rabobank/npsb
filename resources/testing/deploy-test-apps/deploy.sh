set -eu
cf t -o it4it-org -s spaceA
for A in appA appB appC
do
  cf push -f cf-statics/manifest.yml -p cf-statics "$A" --no-start
done
cf t -o it4it-org -s spaceB
for A in appD appE
do
  cf push -f cf-statics/manifest.yml -p cf-statics "$A" --no-start
  cf map-route "$A" apps.internal --hostname "$A"
done
cf t -o it4it-org -s spaceC
for A in appF appG appH
do
  cf push -f cf-statics/manifest.yml -p cf-statics "$A" --no-start
done
cf t -o it4it-org -s spaceD
for A in appI appJ
do
  cf push -f cf-statics/manifest.yml -p cf-statics "$A" --no-start
  cf map-route "$A" apps.internal --hostname "$A"
done
cf t -o it4it-org -s spaceE
for A in appL appM
do
  cf push -f cf-statics/manifest.yml -p cf-statics "$A" --no-start
done
for A in appN appO
do
  cf push -f cf-statics/manifest.yml -p cf-statics "$A" --no-start
  cf map-route "$A" apps.internal --hostname "$A"
done
