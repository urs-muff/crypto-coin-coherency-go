# Check if IPFS is installed
if (-not (Get-Command ipfs -ErrorAction SilentlyContinue)) {
    Write-Error "IPFS is not installed. Please install IPFS and try again."
    exit
}

ipfs files rm /ccn/peer-list.json
ipfs files rm /ccn/relationships.json
ipfs files rm /ccn/conceptID-CID.json
ipfs files rm /ccn/concepts.json
ipfs files rm /ccn/instanceID-CID.json
ipfs files rm /ccn/instances.json

ipfs repo gc

# Get all pinned entries
$pinnedEntries = ipfs pin ls --type=all | ForEach-Object { $_.ToString().Split()[0] }

if ($pinnedEntries.Count -eq 0) {
    ipfs repo ls
    Write-Output "No pinned entries found."
    exit
}

# Unpin all pinned entries
foreach ($entry in $pinnedEntries) {
    Write-Output "Unpinning $entry"
    ipfs pin rm $entry
}

# Garbage collect to clean up the unpinned entries
Write-Output "Running garbage collection..."
ipfs repo gc

Write-Output "All pinned 'ipfs repo' entries have been removed."
ipfs repo ls