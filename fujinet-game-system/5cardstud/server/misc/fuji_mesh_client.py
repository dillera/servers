# WIP Testing file for VICE

import sys
import time
import os
import requests
from pubsub import pub
import meshtastic
import meshtastic.serial_interface

workingPath = "/Users/eric/Documents/projects/vice-device"

receivedChunks = {}

def onReceive(packet, interface):  # pylint: disable=unused-argument
    global receivedChunks
    if packet['decoded']['portnum'] == 'PRIVATE_APP':
        # print(f"Payload: {packet['decoded']['payload']}")
        payload = packet['decoded']['payload']
        print(f"Received {len(payload)} bytes:")
        hexdump(payload)

        receivedChunks[payload[0]] = payload[2:]

        # Have we received all chunks yet? Write everything to a single file
        if len(receivedChunks) == payload[1]:
            with open(outFile, "wb") as file:
                result = b"".join(receivedChunks[k] for k in sorted(receivedChunks.keys()))
                file.write(result)
            if os.path.exists(watchFile):
                os.remove(watchFile)

        

def onConnection(interface, topic=pub.AUTO_TOPIC):  # pylint: disable=unused-argument
    global connected
    connected = True
    print(f"Connected to {interface}")
    

def processWatchFile():
    """
    Read the command of the watch file and send it as a message via the meshtastic interface.
    """

    with open(watchFile, "r", encoding="utf-8") as file:
        command = file.read().strip().lower()
        if command.startswith("n:"):
            command = command[2:]  # remove first two characters ("n:")
        print(f"Sending: {command}")
        payload = command.encode('utf-8')
        if os.path.exists(outFile):
            os.remove(outFile)
        iface.sendData(payload, portNum="PRIVATE_APP")
        #iface.sendText(command)

def hexdump(data: bytes, width: int = 16):
    for i in range(0, len(data), width):
        chunk = data[i:i+width]
        
        # Hex view
        hex_bytes = " ".join(f"{b:02x}" for b in chunk)
        
        # Pad hex output to align text view
        hex_bytes = hex_bytes.ljust(width * 3)
        
        # Text view (printable ASCII, else '.')
        text = "".join(chr(b) if 32 <= b < 127 else "." for b in chunk)
        
        print(f"{i:08x}  {hex_bytes}  {text}")

print("Meshtastic Mock FujiNet Bridge - CLIENT")
print("-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-")
print("Looking for meshtastic devices...")

ports = meshtastic.util.findPorts(True)
if len(ports) == 0:
    print("No meshtastic devices found")
    sys.exit(0)

connected = False
usbPort = "/dev/cu.usbserial-0001"

pub.subscribe(onReceive, "meshtastic.receive")
pub.subscribe(onConnection, "meshtastic.connection.established")

iface = meshtastic.serial_interface.SerialInterface(usbPort)

while connected is False:
    time.sleep(0.1)

watchFile = workingPath + "/vice-out"
outFile = workingPath + "/vice-in"

prevWatchFileCtime = 0
waitMinAmount = 0.050
waitMaxAmount = 0.200
waitAmount = waitMinAmount

print(f"Watching: {watchFile}")

try:
    while True:
        if os.path.exists(watchFile) and os.path.getsize(watchFile) > 0 and os.path.getctime(watchFile) != prevWatchFileCtime:
            try:
                with open(watchFile, 'a') as f:
                    writable = f.writable()
                if writable:
                    prevWatchFileCtime = os.path.getctime(watchFile)
                    waitAmount=waitMinAmount
                    receivedChunks = {}
                    processWatchFile()
                else:
                    waitAmount = min(waitAmount + waitMinAmount, waitMaxAmount) 
            except IOError:
                time.sleep(waitMinAmount)
           

        time.sleep(waitMinAmount)  #  waitMinAmount
except KeyboardInterrupt:
    print("Exiting")
iface.close()
