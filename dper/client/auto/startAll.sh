osascript -e 'tell application "Terminal" to do script "cd /Users/dengmingtao/Documents/GitHub/dpchain/dper/client/auto/dper_booter1 && ./start.sh; exec bash"'
osascript -e 'tell application "Terminal" to do script "cd /Users/dengmingtao/Documents/GitHub/dpchain/dper/client/auto/dper_dper1 && ./start.sh; exec bash"'
osascript -e 'tell application "Terminal" to do script "cd /Users/dengmingtao/Documents/GitHub/dpchain/dper/client/auto/dper_dper2 && ./start.sh; exec bash"'
osascript -e 'tell application "Terminal" to do script "cd /Users/dengmingtao/Documents/GitHub/dpchain/dper/client/auto/dper_dper3 && ./start.sh; exec bash"'
sleep 25
osascript -e 'tell application "Terminal" to do script "cd /Users/dengmingtao/Documents/GitHub/dpchain/dper/client/auto/dper_dper1 && ./run_contract.sh; exec bash"'
osascript -e 'tell application "Terminal" to do script "cd /Users/dengmingtao/Documents/GitHub/dpchain/dper/client/auto/dper_dper2 && ./run_contract.sh; exec bash"'
osascript -e 'tell application "Terminal" to do script "cd /Users/dengmingtao/Documents/GitHub/dpchain/dper/client/auto/dper_dper3 && ./run_contract.sh; exec bash"'