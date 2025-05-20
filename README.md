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
├── cmd/                     # Entry point(s) of the application (e.g., main.go)
├── core/                    # Core blockchain components such as blocks, transactions, and node state management
├── crypto/                  # Cryptographic utilities including hashing, signing, and key management
├── network/                 # Peer-to-peer networking layer responsible for discovery and message propagation
├── consensus/               # Consensus algorithms implementation (e.g., Lachesis, Ethash)
├── api/                     # RESTful or gRPC API server for external interaction with the blockchain
├── explorer/                # (Optional) Blockchain explorer frontend to visualize blocks and transactions
├── scripts/                 # Build, deployment, and automation scripts for development workflows
├── docs/                    # Project documentation including design specs and technical guides
├── tests/                   # Unit and integration tests to ensure code quality and correctness
├── .github/                 # GitHub configuration files such as Actions workflows, issue and PR templates
├── .gitignore               # Specifies intentionally untracked files to ignore in Git
├── README.md                # Project overview, setup instructions, and general information
├── LICENSE                  # Licensing information for the project (e.g., MIT, Apache 2.0)
└── CONTRIBUTING.md          # Guidelines for contributing to the project and code of conduct

