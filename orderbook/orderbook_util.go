package orderbook

// 【新增】輔助函數 - 獲取訂單方向名稱
func GetSideName(side OrderSide) string {
	switch side {
	case Bid:
		return "買單"
	case Ask:
		return "賣單"
	default:
		return "未知方向"
	}
}

// 【新增】輔助函數 - 獲取訂單狀態名稱
func GetStatusName(status OrderStatus) string {
	switch status {
	case Pending:
		return "等待中"
	case Filled:
		return "已成交"
	case Partial:
		return "部分成交"
	case Cancelled:
		return "已取消"
	default:
		return "未知狀態"
	}
}

// 【新增】輔助函數 - 獲取訂單類型名稱
func GetTypeName(orderType OrderType) string {
	switch orderType {
	case Limit:
		return "限價單"
	case Market:
		return "市價單"
	default:
		return "未知類型"
	}
}
