package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	shell "github.com/ipfs/go-ipfs-api"
)

// IPFSShell implements the Node_i interface using go-ipfs-api
type IPFSShell struct {
	sh *shell.Shell
}

func NewIPFSShell(url string) *IPFSShell {
	return &IPFSShell{sh: shell.NewShell(url)}
}

func (i *IPFSShell) Add(ctx context.Context, content io.Reader) (CID, error) {
	cid, err := i.sh.Add(content, shell.CidVersion(1), shell.Pin(true))
	return CID(cid), err
}

func (i *IPFSShell) Get(ctx context.Context, cid CID) (io.ReadCloser, error) {
	return i.sh.Cat(string(cid))
}

func (i *IPFSShell) Remove(ctx context.Context, cid CID) error {
	return i.sh.Unpin(string(cid))
}

func (i *IPFSShell) List(ctx context.Context) ([]CID, error) {
	pins, err := i.sh.Pins()
	if err != nil {
		return nil, err
	}
	ret := make([]CID, 0)
	for cid, pinInfo := range pins {
		fmt.Printf("CID:%s, Type=%s\n", cid, pinInfo.Type)
		ret = append(ret, CID(cid))
	}
	return ret, nil
}

func (i *IPFSShell) Publish(ctx context.Context, topic string, data []byte) error {
	return i.sh.PubSubPublish(topic, string(data))
}

func (i *IPFSShell) Subscribe(ctx context.Context, topic string) (<-chan []byte, error) {
	sub, err := i.sh.PubSubSubscribe(topic)
	if err != nil {
		return nil, err
	}

	ch := make(chan []byte)
	go func() {
		defer close(ch)
		for {
			msg, err := sub.Next()
			if err != nil {
				if err == context.Canceled {
					return
				}
				log.Printf("Error receiving message: %v", err)
				continue
			}
			ch <- msg.Data
		}
	}()

	return ch, nil
}

func (i *IPFSShell) Connect(ctx context.Context, peerID PeerID) error {
	return i.sh.SwarmConnect(ctx, string(peerID))
}

func (i *IPFSShell) ListPeers(ctx context.Context) ([]Peer_i, error) {
	swarmPeers, err := i.sh.SwarmPeers(ctx)
	if err != nil {
		return nil, err
	}

	peers := make([]Peer_i, len(swarmPeers.Peers))
	for j, p := range swarmPeers.Peers {
		peers[j] = &Peer{
			ID:        PeerID(p.Peer),
			Timestamp: time.Now(),
		}
	}

	return peers, nil
}

func (i *IPFSShell) Bootstrap(ctx context.Context) error {
	bootstrapNodes := []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	}

	for _, addr := range bootstrapNodes {
		if err := i.sh.SwarmConnect(ctx, addr); err != nil {
			log.Printf("Failed to connect to bootstrap node %s: %v", addr, err)
			// } else {
			//   log.Printf("Connected to bootstrap node: %s", addr)
		}
	}

	return nil
}

func (i *IPFSShell) ID(ctx context.Context) (PeerID, error) {
	info, err := i.sh.ID()
	if err != nil {
		return "", err
	}
	return PeerID(info.ID), nil
}

func (i *IPFSShell) Load(ctx context.Context, path string, target any) error {
	data, err := i.sh.FilesRead(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to read file from IPFS: %v", err)
	}
	defer data.Close()

	if err := json.NewDecoder(data).Decode(target); err != nil {
		return fmt.Errorf("failed to decode data: %v", err)
	}

	log.Printf("Loaded data from IPFS path: %s", path)
	return nil
}

func (i *IPFSShell) Save(ctx context.Context, path string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	if err := i.sh.FilesWrite(ctx, path, bytes.NewReader(jsonData), shell.FilesWrite.Create(true), shell.FilesWrite.Truncate(true), shell.FilesWrite.Parents(true)); err != nil {
		return fmt.Errorf("failed to write file to IPFS: %v", err)
	}

	// log.Printf("Saved data to IPFS path: %s", path)
	return nil
}
