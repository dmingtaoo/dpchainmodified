cd ./dper_booter1
gnome-terminal -t '"dper_booter1"' -- bash -c "./start.sh;exec bash"
cd ..
cd ./dper_dper1
gnome-terminal -t '"dper_dper1" ' -- bash -c "./start.sh;exec bash"
cd ..
cd ./dper_dper2
gnome-terminal -t '"dper_dper2" ' -- bash -c "./start.sh;exec bash"
cd ..
cd ./dper_dper3
gnome-terminal -t '"dper_dper3" ' -- bash -c "./start.sh;exec bash"
cd ..
cd ./dper_dper4
gnome-terminal -t '"dper_dper4" ' -- bash -c "./start.sh;exec bash"
cd ..
sleep 25 
cd ./dper_dper1
gnome-terminal -t '"dper_dper1"' -- bash -c "./run_contract.sh;exec bash"
cd ..
cd ./dper_dper2
gnome-terminal -t '"dper_dper2"' -- bash -c "./run_contract.sh;exec bash"
cd ..
cd ./dper_dper3
gnome-terminal -t '"dper_dper3"' -- bash -c "./run_contract.sh;exec bash"
cd ..
cd ./dper_dper4
gnome-terminal -t '"dper_dper4"' -- bash -c "./run_contract.sh;exec bash"
cd ..
