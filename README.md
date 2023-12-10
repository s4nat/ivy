# Fault Tolerant IVY Protocol (Integrated shared Virtual memory at Yale) 

Go-based implementation of fault-tolerant Ivy Architecture with a backup central manager for consistent (meta)data handling during primary CM failures.

# How to run the code
There are 2 types of nodes, Central Manager and Client. Each node runs on separate terminals on separate processes and communicate with each other using RPC calls. For a fresh run of the protocol, ensure the `/data` folder is empty.

## Running the different types of Nodes
To start the primary CM:
1. Ensure you are in the root directory of the project.
2. Run `go build && ./ivy`
3. You will be prompted to choose the node type: "Enter Node type ('1': CM, '2': Client, 'restartCM', 'restartBackup')"
4. Type '1'. This will create a new `cm.json` file and add the primary CM object to the file.
5. The primary CM should now be running.

To start the Backup CM:
1. Run `go build && ./ivy`
2. You will be prompted to choose the node type: "Enter Node type ('1': CM, '2': Client, 'restartCM', 'restartBackup')"
3. Type '1'. This will see that a primary CM already exists in `cm.json ` and add a Backup CM object to the file.
4. The Backup CM should now be running.

To start a Client:
1. Run `go build && ./ivy`
2. You will be prompted to choose the node type: "Enter Node type ('1': CM, '2': Client, 'restartCM', 'restartBackup')"
3. Type '1'. This will check `client.json ` and add Client (currentHighestID + 1) to the file.
4. The Client should now be running

## How to kill any Node (PrimaryCM/BackupCM/Client)
To kill any node simply go to its terminal and press `ctrl+c`

## How to reboot a CM (PrimaryCM/BackupCM)
To reboot a PrimaryCM:
1. Run `go build && ./ivy`
2. You will be prompted to choose the node type: "Enter Node type ('1': CM, '2': Client, 'restartCM', 'restartBackup')"
3. Type 'restartCM'. This checks `cm.json` for the IP of the original Primary CM and runs the CM on that IP.

To reboot a BackupCM:
1. Run `go build && ./ivy`
2. You will be prompted to choose the node type: "Enter Node type ('1': CM, '2': Client, 'restartCM', 'restartBackup')"
3. Type 'restartBackupCM'. This checks `cm.json` for the IP of the Backup CM and runs the CM on that IP.

## How to send read/write requests
- Send a write request by typing `writePage <pageNo> <content>`. For example, you type `writePage P1 Content1`.
- Send a read request by typinh  `readPage <pageNo>`. For example, you type `readPage P1`.

## Useful command
- CM: Type `print` to view the MetaData.
- Client: Type `print` to view the PageStore

# Fault Tolerance

## Syncing MetaData
When you have 2 CMs running: one Primary CM and one Backup CM, the Backup CM has runs goroutine (`go pulseCheck()`). With this, it polls the Primary CM every 1s to check if it is alive. In response, the Primary CM sends a Payload containing its MetaData. This way, every second the Backup CM has synced up with the Primary CM.

## Transfer of Primary title
When the Primary CM is killed, the pulseCheck will detect the death of the Primary CM. The Backup CM now takes over as Primary and sends a `CHANGE_CM` message to all Clients in `client.json` to notify them about the change in CM. All read and write requests are now routed to the Backup CM.

## Rebooting Primary CM
When you reboot the Primary CM, it sends a `IM_BACK` message to the Backup CM to let them know who's the real boss. It again sends a `CHANGE_CM` message to all the Clients to inform them about the change in CM. Read/Write requests are back to being routed to the Primary CM.

## Rebooting Backup CM
In the event of Backup CM death, you can reboot it. This will simply restart the goroutine to send `PulseChecks` to the PrimaryCM and resume syncing the MetaData every 1s.


#Test results
