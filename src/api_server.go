package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// APIServer سرور API برای dApp مشابه Fantom Opera
type APIServer struct {
	consensusEngine *ConsensusEngine
	networkManager  *NetworkManager
	stateDB         *StateDB
	port            string
	upgrader        websocket.Upgrader

	// Rate limiting
	rateLimiter *RateLimiter

	// API versioning
	version string
}

// RateLimiter محدودیت نرخ درخواست
type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
}

// APIResponse پاسخ استاندارد API مشابه Fantom Opera
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Version string      `json:"version,omitempty"`
}

// ValidatorInfo اطلاعات validator برای API
type ValidatorInfo struct {
	Address    string  `json:"address"`
	Stake      string  `json:"stake"`
	IsActive   bool    `json:"is_active"`
	Commission uint64  `json:"commission"`
	Rewards    string  `json:"rewards"`
	Score      float64 `json:"score"`
}

// StakingInfo اطلاعات staking
type StakingInfo struct {
	TotalStake       string `json:"total_stake"`
	ActiveValidators int    `json:"active_validators"`
	MinStake         string `json:"min_stake"`
	YourStake        string `json:"your_stake,omitempty"`
	APY              string `json:"apy,omitempty"`
}

// NetworkInfo اطلاعات شبکه
type NetworkInfo struct {
	NetworkName     string `json:"network_name"`
	Consensus       string `json:"consensus"`
	LatestBlock     uint64 `json:"latest_block"`
	BlockTime       string `json:"block_time"`
	TotalValidators int    `json:"total_validators"`
	TotalStake      string `json:"total_stake"`
	GasPrice        string `json:"gas_price"`
	ChainID         string `json:"chain_id"`
}

// TransactionInfo اطلاعات تراکنش
type TransactionInfo struct {
	Hash      string `json:"hash"`
	From      string `json:"from"`
	To        string `json:"to"`
	Value     string `json:"value"`
	GasUsed   uint64 `json:"gas_used"`
	GasPrice  string `json:"gas_price"`
	Status    string `json:"status"`
	BlockNum  uint64 `json:"block_number"`
	Timestamp uint64 `json:"timestamp"`
}

// BlockInfo اطلاعات بلاک
type BlockInfo struct {
	Number       uint64   `json:"number"`
	Hash         string   `json:"hash"`
	ParentHash   string   `json:"parent_hash"`
	Timestamp    uint64   `json:"timestamp"`
	GasUsed      uint64   `json:"gas_used"`
	GasLimit     uint64   `json:"gas_limit"`
	Transactions []string `json:"transactions"`
	Validator    string   `json:"validator"`
}

// NewAPIServer ایجاد API server جدید
func NewAPIServer(consensus *ConsensusEngine, network *NetworkManager, state *StateDB, port string) *APIServer {
	return &APIServer{
		consensusEngine: consensus,
		networkManager:  network,
		stateDB:         state,
		port:            port,
		version:         "1.0.0",
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // برای توسعه، در production باید محدود شود
			},
		},
		rateLimiter: &RateLimiter{
			requests: make(map[string][]time.Time),
		},
	}
}

// Start شروع API server
func (api *APIServer) Start() error {
	router := mux.NewRouter()

	// CORS middleware
	router.Use(api.corsMiddleware)

	// Rate limiting middleware
	router.Use(api.rateLimitMiddleware)

	// API routes
	api.setupRoutes(router)

	fmt.Printf("🌐 API Server starting on port %s\n", api.port)
	return http.ListenAndServe(":"+api.port, router)
}

// setupRoutes تنظیم routes مشابه Fantom Opera
func (api *APIServer) setupRoutes(router *mux.Router) {
	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()

	// Blockchain info
	v1.HandleFunc("/blockchain/info", api.getBlockchainInfo).Methods("GET")
	v1.HandleFunc("/blockchain/blocks", api.getBlocks).Methods("GET")
	v1.HandleFunc("/blockchain/blocks/{number}", api.getBlock).Methods("GET")
	v1.HandleFunc("/blockchain/transactions/{hash}", api.getTransaction).Methods("GET")

	// Validator management
	v1.HandleFunc("/validators", api.getValidators).Methods("GET")
	v1.HandleFunc("/validators/register", api.registerValidator).Methods("POST")
	v1.HandleFunc("/validators/unregister", api.unregisterValidator).Methods("POST")
	v1.HandleFunc("/validators/stake", api.updateStake).Methods("POST")

	// Staking
	v1.HandleFunc("/staking/info", api.getStakingInfo).Methods("GET")
	v1.HandleFunc("/staking/delegate", api.delegate).Methods("POST")
	v1.HandleFunc("/staking/undelegate", api.undelegate).Methods("POST")

	// Account management
	v1.HandleFunc("/accounts/{address}", api.getAccount).Methods("GET")
	v1.HandleFunc("/accounts/{address}/balance", api.getBalance).Methods("GET")
	v1.HandleFunc("/accounts/{address}/transactions", api.getTransactions).Methods("GET")

	// Network
	v1.HandleFunc("/network/peers", api.getPeers).Methods("GET")
	v1.HandleFunc("/network/stats", api.getNetworkStats).Methods("GET")

	// EVM endpoints
	v1.HandleFunc("/evm/contracts/{address}", api.getContract).Methods("GET")
	v1.HandleFunc("/evm/contracts/{address}/call", api.callContract).Methods("POST")
	v1.HandleFunc("/evm/contracts/deploy", api.deployContract).Methods("POST")

	// SINAR Token endpoints
	v1.HandleFunc("/sinar/info", api.getSINARInfo).Methods("GET")
	v1.HandleFunc("/sinar/balance/{address}", api.getSINARBalance).Methods("GET")
	v1.HandleFunc("/sinar/transfer", api.transferSINAR).Methods("POST")
	v1.HandleFunc("/sinar/price", api.getSINARPrice).Methods("GET")
	v1.HandleFunc("/sinar/supply", api.getSINARSupply).Methods("GET")
	v1.HandleFunc("/sinar/distribution", api.getSINARDistribution).Methods("GET")
	v1.HandleFunc("/sinar/initialize", api.initializeSINAR).Methods("POST")

	// WebSocket for real-time updates
	v1.HandleFunc("/ws", api.handleWebSocket)

	// Mobile app specific endpoints
	v1.HandleFunc("/mobile/validator/setup", api.setupMobileValidator).Methods("POST")
	v1.HandleFunc("/mobile/validator/status", api.getMobileValidatorStatus).Methods("GET")
	v1.HandleFunc("/mobile/staking/quick", api.quickStaking).Methods("POST")
}

// corsMiddleware CORS middleware
func (api *APIServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// rateLimitMiddleware Rate limiting middleware
func (api *APIServer) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr

		api.rateLimiter.mu.Lock()
		now := time.Now()
		requests := api.rateLimiter.requests[clientIP]

		// حذف درخواست‌های قدیمی (بیش از 1 دقیقه)
		var validRequests []time.Time
		for _, reqTime := range requests {
			if now.Sub(reqTime) < time.Minute {
				validRequests = append(validRequests, reqTime)
			}
		}

		// بررسی محدودیت (حداکثر 100 درخواست در دقیقه)
		if len(validRequests) >= 100 {
			api.sendErrorResponse(w, "Rate limit exceeded", http.StatusTooManyRequests)
			api.rateLimiter.mu.Unlock()
			return
		}

		// اضافه کردن درخواست جدید
		validRequests = append(validRequests, now)
		api.rateLimiter.requests[clientIP] = validRequests
		api.rateLimiter.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// getBlockchainInfo اطلاعات کلی بلاکچین
func (api *APIServer) getBlockchainInfo(w http.ResponseWriter, r *http.Request) {
	latestBlock := api.consensusEngine.GetLatestBlock()

	var latestBlockNum uint64
	if latestBlock != nil {
		latestBlockNum = latestBlock.Header.Number
	}

	info := NetworkInfo{
		NetworkName:     "Sinar Chain",
		Consensus:       "Lachesis (aBFT)",
		LatestBlock:     latestBlockNum,
		BlockTime:       "2 seconds",
		TotalValidators: len(api.consensusEngine.GetValidators()),
		TotalStake:      api.stateDB.GetTotalStake().String(),
		GasPrice:        "1",
		ChainID:         "250",
	}

	api.sendResponse(w, info, "")
}

// getValidators لیست validators
func (api *APIServer) getValidators(w http.ResponseWriter, r *http.Request) {
	validators := api.consensusEngine.GetValidators()

	var validatorInfos []ValidatorInfo
	for _, validator := range validators {
		validatorInfos = append(validatorInfos, ValidatorInfo{
			Address:    validator.Address,
			Stake:      fmt.Sprintf("%d", validator.Stake),
			IsActive:   validator.IsActive,
			Commission: 1000, // 10%
			Rewards:    "0",
			Score:      1.0,
		})
	}

	api.sendResponse(w, validatorInfos, "")
}

// registerValidator ثبت validator جدید
func (api *APIServer) registerValidator(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Address    string `json:"address"`
		Stake      string `json:"stake"`
		Commission uint64 `json:"commission"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		api.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// تبدیل stake به عدد
	stake, err := strconv.ParseUint(request.Stake, 10, 64)
	if err != nil {
		api.sendErrorResponse(w, "Invalid stake amount", http.StatusBadRequest)
		return
	}

	// ایجاد validator
	privKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	validator := NewValidator(request.Address, privKey, stake)

	// اضافه کردن به consensus engine
	if err := api.consensusEngine.AddValidator(validator); err != nil {
		api.sendErrorResponse(w, fmt.Sprintf("Failed to register validator: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"validator_address": request.Address,
		"stake":             request.Stake,
		"status":            "registered",
	}

	api.sendResponse(w, response, "")
}

// getStakingInfo اطلاعات staking
func (api *APIServer) getStakingInfo(w http.ResponseWriter, r *http.Request) {
	validators := api.consensusEngine.GetValidators()
	activeValidators := 0
	for _, v := range validators {
		if v.IsActive {
			activeValidators++
		}
	}

	info := StakingInfo{
		TotalStake:       api.stateDB.GetTotalStake().String(),
		ActiveValidators: activeValidators,
		MinStake:         "1000000", // 1M tokens
		APY:              "8.5",     // 8.5% APY
	}

	api.sendResponse(w, info, "")
}

// delegate delegation
func (api *APIServer) delegate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Delegator string `json:"delegator"`
		Validator string `json:"validator"`
		Amount    string `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		api.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseUint(request.Amount, 10, 64)
	if err != nil {
		api.sendErrorResponse(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	// انجام delegation
	delegatorAddr := common.HexToAddress(request.Delegator)
	validatorAddr := common.HexToAddress(request.Validator)

	if err := api.stateDB.Delegate(delegatorAddr, validatorAddr, big.NewInt(int64(amount))); err != nil {
		api.sendErrorResponse(w, fmt.Sprintf("Delegation failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"delegator": request.Delegator,
		"validator": request.Validator,
		"amount":    request.Amount,
		"status":    "delegated",
	}

	api.sendResponse(w, response, "")
}

// setupMobileValidator راه‌اندازی validator موبایل
func (api *APIServer) setupMobileValidator(w http.ResponseWriter, r *http.Request) {
	var request struct {
		DeviceID   string     `json:"device_id"`
		Stake      string     `json:"stake"`
		DeviceInfo DeviceInfo `json:"device_info"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		api.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// ایجاد کلید خصوصی برای validator موبایل
	privKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	address := crypto.PubkeyToAddress(privKey.PublicKey)

	stake, err := strconv.ParseUint(request.Stake, 10, 64)
	if err != nil {
		api.sendErrorResponse(w, "Invalid stake amount", http.StatusBadRequest)
		return
	}

	// ایجاد validator
	validator := NewValidator(request.DeviceID, privKey, stake)

	// اضافه کردن به consensus engine
	if err := api.consensusEngine.AddValidator(validator); err != nil {
		api.sendErrorResponse(w, fmt.Sprintf("Failed to setup mobile validator: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"validator_address": address.Hex(),
		"private_key":       fmt.Sprintf("%x", privKey.D.Bytes()),
		"stake":             request.Stake,
		"device_id":         request.DeviceID,
		"status":            "active",
	}

	api.sendResponse(w, response, "")
}

// getMobileValidatorStatus بررسی وضعیت validator موبایل
func (api *APIServer) getMobileValidatorStatus(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		api.sendErrorResponse(w, "Device ID is required", http.StatusBadRequest)
		return
	}

	validators := api.consensusEngine.GetValidators()
	var validatorStatus map[string]interface{}

	for _, validator := range validators {
		if validator.ID == deviceID {
			validatorStatus = map[string]interface{}{
				"device_id":         deviceID,
				"validator_address": validator.Address,
				"stake":             fmt.Sprintf("%d", validator.Stake),
				"is_active":         validator.IsActive,
				"last_seen":         validator.LastSeen.Format(time.RFC3339),
				"status":            "active",
			}
			break
		}
	}

	if validatorStatus == nil {
		validatorStatus = map[string]interface{}{
			"device_id": deviceID,
			"status":    "not_found",
		}
	}

	api.sendResponse(w, validatorStatus, "")
}

// quickStaking staking سریع
func (api *APIServer) quickStaking(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Amount string `json:"amount"`
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		api.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseUint(request.Amount, 10, 64)
	if err != nil {
		api.sendErrorResponse(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	// انجام staking سریع
	response := map[string]interface{}{
		"user_id":           request.UserID,
		"amount":            request.Amount,
		"stake_id":          fmt.Sprintf("stake_%s_%d", request.UserID, time.Now().Unix()),
		"status":            "staked",
		"apy":               "8.5",
		"estimated_rewards": fmt.Sprintf("%.2f", float64(amount)*0.085),
	}

	api.sendResponse(w, response, "")
}

// handleWebSocket مدیریت WebSocket
func (api *APIServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := api.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// ارسال پیام خوش‌آمدگویی
	welcomeMsg := map[string]interface{}{
		"type":    "welcome",
		"message": "Connected to Sinar Chain WebSocket",
		"version": api.version,
	}
	conn.WriteJSON(welcomeMsg)

	// حلقه دریافت پیام‌ها
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		// پردازش پیام
		api.handleWebSocketMessage(conn, msg)
	}
}

// handleWebSocketMessage پردازش پیام WebSocket
func (api *APIServer) handleWebSocketMessage(conn *websocket.Conn, msg map[string]interface{}) {
	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "subscribe_blocks":
		// ارسال اطلاعات بلاک‌های جدید
		latestBlock := api.consensusEngine.GetLatestBlock()
		if latestBlock != nil {
			blockInfo := BlockInfo{
				Number:     latestBlock.Header.Number,
				Hash:       latestBlock.Hash().Hex(),
				ParentHash: latestBlock.Header.ParentHash.Hex(),
				Timestamp:  uint64(latestBlock.Header.CreatedAt.Unix()),
				GasUsed:    latestBlock.Header.GasUsed,
				GasLimit:   latestBlock.Header.GasLimit,
				Validator:  latestBlock.Header.Creator,
			}

			response := map[string]interface{}{
				"type": "new_block",
				"data": blockInfo,
			}
			conn.WriteJSON(response)
		}

	case "subscribe_transactions":
		// ارسال اطلاعات تراکنش‌های جدید
		response := map[string]interface{}{
			"type": "new_transaction",
			"data": "Transaction data will be sent here",
		}
		conn.WriteJSON(response)

	case "get_network_stats":
		// ارسال آمار شبکه
		networkStats := api.networkManager.GetNetworkStats()
		response := map[string]interface{}{
			"type": "network_stats",
			"data": networkStats,
		}
		conn.WriteJSON(response)
	}
}

// sendResponse ارسال پاسخ موفق
func (api *APIServer) sendResponse(w http.ResponseWriter, data interface{}, err string) {
	response := APIResponse{
		Success: err == "",
		Data:    data,
		Error:   err,
		Version: api.version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendErrorResponse ارسال پاسخ خطا
func (api *APIServer) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := APIResponse{
		Success: false,
		Error:   message,
		Version: api.version,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// DeviceInfo اطلاعات دستگاه
type DeviceInfo struct {
	Platform string `json:"platform"`
	Version  string `json:"version"`
	Model    string `json:"model"`
}

// Placeholder methods for other endpoints
func (api *APIServer) getBlocks(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, []BlockInfo{}, "")
}

func (api *APIServer) getBlock(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, BlockInfo{}, "")
}

func (api *APIServer) getTransaction(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, TransactionInfo{}, "")
}

func (api *APIServer) unregisterValidator(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, map[string]string{"status": "unregistered"}, "")
}

func (api *APIServer) updateStake(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, map[string]string{"status": "updated"}, "")
}

func (api *APIServer) undelegate(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, map[string]string{"status": "undelegated"}, "")
}

func (api *APIServer) getAccount(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, map[string]interface{}{"address": "0x..."}, "")
}

func (api *APIServer) getBalance(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, map[string]string{"balance": "0"}, "")
}

func (api *APIServer) getTransactions(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, []TransactionInfo{}, "")
}

func (api *APIServer) getPeers(w http.ResponseWriter, r *http.Request) {
	peers := api.networkManager.GetPeers()
	api.sendResponse(w, peers, "")
}

func (api *APIServer) getNetworkStats(w http.ResponseWriter, r *http.Request) {
	stats := api.networkManager.GetNetworkStats()
	api.sendResponse(w, stats, "")
}

func (api *APIServer) getContract(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, map[string]interface{}{"address": "0x..."}, "")
}

func (api *APIServer) callContract(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, map[string]string{"result": "0x..."}, "")
}

func (api *APIServer) deployContract(w http.ResponseWriter, r *http.Request) {
	api.sendResponse(w, map[string]string{"address": "0x..."}, "")
}

// getSINARInfo دریافت اطلاعات ارز سینار
func (api *APIServer) getSINARInfo(w http.ResponseWriter, r *http.Request) {
	info := api.stateDB.GetSINARInfo()
	api.sendResponse(w, info, "")
}

// getSINARBalance دریافت موجودی سینار
func (api *APIServer) getSINARBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	addr := common.HexToAddress(address)
	balance := api.stateDB.GetSINARBalance(addr)

	response := map[string]interface{}{
		"address": address,
		"balance": balance.String(),
		"symbol":  "SINAR",
	}

	api.sendResponse(w, response, "")
}

// transferSINAR انتقال سینار
func (api *APIServer) transferSINAR(w http.ResponseWriter, r *http.Request) {
	var request struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount string `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		api.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	from := common.HexToAddress(request.From)
	to := common.HexToAddress(request.To)
	amount, ok := new(big.Int).SetString(request.Amount, 10)
	if !ok {
		api.sendErrorResponse(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	if err := api.stateDB.TransferSINAR(from, to, amount); err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"from":    request.From,
		"to":      request.To,
		"amount":  amount.String(),
	}

	api.sendResponse(w, response, "")
}

// getSINARPrice دریافت قیمت سینار
func (api *APIServer) getSINARPrice(w http.ResponseWriter, r *http.Request) {
	price := api.stateDB.GetSINARPrice()

	response := map[string]interface{}{
		"price_usd": price.String(),
		"symbol":    "SINAR",
	}

	api.sendResponse(w, response, "")
}

// getSINARSupply دریافت اطلاعات supply سینار
func (api *APIServer) getSINARSupply(w http.ResponseWriter, r *http.Request) {
	info := api.stateDB.GetSINARInfo()

	response := map[string]interface{}{
		"total_supply":   info["total_supply"],
		"current_supply": info["current_supply"],
		"symbol":         "SINAR",
	}

	api.sendResponse(w, response, "")
}

// getSINARDistribution دریافت اطلاعات توزیع سینار
func (api *APIServer) getSINARDistribution(w http.ResponseWriter, r *http.Request) {
	distribution := api.stateDB.GetDistributionInfo()
	api.sendResponse(w, distribution, "")
}

// initializeSINAR توزیع اولیه سینار
func (api *APIServer) initializeSINAR(w http.ResponseWriter, r *http.Request) {
	if err := api.stateDB.InitializeSINARDistribution(); err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "SINAR distribution initialized successfully",
	}

	api.sendResponse(w, response, "")
}
