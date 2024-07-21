# Crypto Coherency Network: Go Backend

## Project Summary

The Crypto Coherency Network is a decentralized platform designed to manage and interact with various concepts and relationships using a peer-to-peer network. The backend is implemented in Go and leverages IPFS for content addressing and distribution.

### Key Features
- **Concept Management**: Create, update, and manage concepts and their relationships.
- **Peer Management**: Handle peer interactions and maintain a coherent network state.
- **IPFS Integration**: Utilize IPFS for decentralized storage and retrieval of data.
- **WebSocket Communication**: Real-time updates and interactions using WebSockets.

## Setup Instructions

### Prerequisites
- Go 1.16 or higher
- IPFS daemon running
- Make sure you have `gin` and `websocket` packages installed

### Installation

1. **Clone the repository**
    ```sh
    git clone https://github.com/yourusername/crypto-coherency-network.git
    cd crypto-coherency-network
    ```

2. **Install dependencies**
    ```sh
    go mod tidy
    ```

3. **Run the IPFS daemon**
    ```sh
    ipfs daemon
    ```

4. **Build and run the application**
    ```sh
    go build -o crypto-coherency-network
    ./crypto-coherency-network
    ```

### Configuration

Ensure that your IPFS daemon is running and accessible. You can configure the IPFS settings in the `ipfs-shell.go` file if needed.

### Usage

Once the application is running, you can interact with it via the provided API endpoints or through the WebSocket interface for real-time updates.

## Contributing

We welcome contributions! Please fork the repository and submit pull requests.

## License

This project is licensed under the MIT License.
