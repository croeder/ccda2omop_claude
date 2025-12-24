
for file in  ../CCDA-data/xml_load_test/*.xml
do
base_file=$(basename $file | cut -f 1 -d \. )
echo $base_file
mkdir output/$base_file 2> /dev/null
./ccda2omop -input $file  -output  output/$base_file
done

for dir in output/*
do
    cat $dir/condition_occurrence.csv >> condition_occurrence.csv
    cat $dir/device_exposure.csv >> device_exposure.csv
    cat $dir/drug_exposure.csv >> drug_exposure.csv
    cat $dir/measurement.csv >> measurement.csv
    cat $dir/observation.csv >> observation.csv
    cat $dir/person.csv >> person.csv
    cat $dir/procedure_occurrence.csv >> procedure_occurrence.csv
    cat $dir/visit_occurrence.csv >> visit_occurrence.csv
done

wc -l *.csv
