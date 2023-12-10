# Fault Tolerant IVY Protocol (Integrated shared Virtual memory at Yale) 

Go-based implementation of fault-tolerant Ivy Architecture with a backup central manager for consistent (meta)data handling during primary CM failures.

# How to run the code
There are 2 types of nodes, Central Manager and Client. Each node runs on separate terminals on separate processes and communicate with each other using RPC calls. For a fresh run of the protocol, ensure the `/data` folder is empty.

To start the primary CM:
1. Ensure you are in the root directory of the project.
2. run `go build && ./ivy`
3. You will be prompted to choose the node type: "Enter Node type ('1': CM, '2': Client, 'restartCM', 'restartBackup')"
4. Choose '1'. This will create a new `cm.json` file and add the primary CM object to the file.
5.The primary CM should now be running.

To start the Backup CM:
1. run `go build && ./ivy`
2. You will be prompted to choose the node type: "Enter Node type ('1': CM, '2': Client, 'restartCM', 'restartBackup')"
3. Choose '1'. This will see that a primary CM already exists in `cm.json ` and add a Backup CM object to the file.
4. The Backup CM should now be running.

To start a Client:
1. run `go build && ./ivy`
2. You will be prompted to choose the node type: "Enter Node type ('1': CM, '2': Client, 'restartCM', 'restartBackup')"
3. Choose '1'. This will check `client.json ` and add Client (currentHighestID + 1) to the file.
4. The Client should now be running

