# Check if IPFS is installed
if (-not (Get-Command ipfs -ErrorAction SilentlyContinue)) {
    Write-Error "IPFS is not installed. Please install IPFS and try again."
    exit
}

# Get all pinned entries
$pinnedEntries = ipfs pin ls --type=all | ForEach-Object { $_.ToString().Split()[0] }

if ($pinnedEntries.Count -eq 0) {
    Write-Output "No pinned entries found."
    exit
}

# Unpin all pinned entries
foreach ($entry in $pinnedEntries) {
    Write-Output "Unpinning $entry"
    ipfs pin rm $entry
}

ipfs files rm /ccn/relationships.json

# Garbage collect to clean up the unpinned entries
Write-Output "Running garbage collection..."
ipfs repo gc

Write-Output "All pinned 'ipfs repo' entries have been removed."
ipfs repo ls