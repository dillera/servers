# WIP Testing file

import sys
import time
import math
import os
import requests
from pubsub import pub

import meshtastic
import meshtastic.serial_interface


def onReceive(packet, interface):  # pylint: disable=unused-argument
    if packet['decoded']['portnum'] == 'PRIVATE_APP':
        # Get url to call
        url = packet['decoded']['payload'].decode('utf-8')

        # Download url result
        result = download_url_as_bytes(url)
        
        print(f"GET {url} ({len(result)})")
        hexdump(result)

        # Send result back to caller in chunks of 200 bytes
        chunk_size = 198
        chunks = math.ceil(len(result)/chunk_size)
        if chunks == 0:
            payload = bytes([1,1])
            print("Sending empty chunk")
            iface.sendData(payload, portNum="PRIVATE_APP")
        else:
            chunk = 1
            for i in range(0, len(result), chunk_size):
                if chunk > 1:
                    time.sleep(0.1)  # small delay between chunks

                payload = bytes([chunk,chunks]) + result[i:i + chunk_size]
                print(f"Sending chunk {chunk}/{chunks}: payload length={len(payload)-2}")

                iface.sendData(payload, portNum="PRIVATE_APP")
                chunk+=1

def onConnection(interface, topic=pub.AUTO_TOPIC):  # pylint: disable=unused-argument
    global connected
    connected = True
    print(f"Connected to {interface}")
    
def download_url_as_bytes(url):
    """
    Download the content from a given URL and return it as a byte array.
    """
    try:
        response = requests.get(url, headers = {"User-Agent": "MockFujiNetBridge/1.0","Accept": "application/json"})
        response.raise_for_status()  # Raise an exception for bad status codes (4xx or 5xx)
        return response.content
    except requests.exceptions.RequestException as e:
        print(f"Error downloading URL: {e}")
        return bytes()

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


print("Meshtastic Mock FujiNet Bridge - SERVER")
print("-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-")
print("Looking for meshtastic devices...")

ports = meshtastic.util.findPorts(True)
if len(ports) == 0:
    print("No meshtastic devices found")
    sys.exit(0)

connected = False
usbPort = "/dev/cu.usbserial-2"

pub.subscribe(onReceive, "meshtastic.receive")
pub.subscribe(onConnection, "meshtastic.connection.established")

iface = meshtastic.serial_interface.SerialInterface(usbPort)

while connected is False:
    time.sleep(0.1)


print("Listening for command from meshtastic..")

try:
    while True:
        time.sleep(1) 
except KeyboardInterrupt:
    print("Exiting")
iface.close()
