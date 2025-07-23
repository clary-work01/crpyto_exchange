# build Go 的編譯命令
# -o - output 參數
# Go 會編譯當前目錄中的 .go 文件 
# 在 bin/ 目錄下生成名為 exchange 的可執行文件 
# 如果 bin/ 目錄不存在，Go 會自動創建它
build:
	go build -o bin/exchange

run: build
	./bin/exchange

# 在當前目錄(./)及其所有子目錄(...)中運行所有測試，並顯示詳細輸出
test:
	go test -v ./...