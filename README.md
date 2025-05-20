# Sinar_Chain
A high-performance blockchain protocol inspired by Ethereum and Fantom.

# 🌐 SinarChain

> A blazing-fast, modular, and developer-friendly blockchain protocol — inspired by the best of Ethereum and Fantom.

![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)
![Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)
![Stage: In Development](https://img.shields.io/badge/status-in--development-yellow)
![Contributions Welcome](https://img.shields.io/badge/contributions-welcome-blue)

---

## 🚀 About SinarChain

**SinarChain** is a next-generation Layer-1 blockchain protocol written in **Go**, built from the ground up with speed, scalability, and modularity in mind.

We are combining proven mechanisms from **Ethereum (EVM, PoS/PoW)** and **Fantom (aBFT, DAG)** to design a flexible and secure decentralized infrastructure optimized for high-throughput and low-latency transactions.

> “Sinar” (سینَر) means *ray* or *beam* — our vision is to beam the next light of decentralization across the world.

---

## ⚙️ Features (WIP)

- ✅ Modular blockchain architecture
- ✅ Fast & secure P2P network
- ✅ EVM-compatible virtual machine (planned)
- ✅ DAG-based consensus core (like Fantom's Lachesis)
- ✅ Hybrid PoS / PoA consensus
- ✅ Smart contract support (Solidity-compatible roadmap)
- ✅ RESTful API and CLI tools
- ✅ Developer-friendly SDK
- 🔒 Cryptographic primitives (ECDSA, keccak256, etc.)

---

## 📂 Project Structure

```bash
sinar-chain/
├── cmd/               # Entry points (node, cli, etc.)
├── core/              # Blockchain logic (blocks, tx, state)
├── consensus/         # PoS / DAG consensus engine
├── crypto/            # Hashing, signing, keypairs
├── network/           # P2P messaging & discovery
├── api/               # REST/gRPC interfaces
├── tests/             # Unit & integration tests
├── scripts/           # Build, deploy, utils
├── docs/              # Documentation
└── README.md
