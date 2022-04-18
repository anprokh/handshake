# handshake

Implementation of the **handshake** procedure performed during node initialization, in accordance with the bitcoin protocol. This is the first thing to do after establishing a network connection between two nodes.
When a node creates an outgoing connection, it will immediately advertise its version. The remote node will respond with its version. No further communication is possible until both peers have exchanged their version. If a “version” message is accepted, the receiving node should send a “verack” message.

![handshake_](https://user-images.githubusercontent.com/103370385/163834737-339389cf-9b08-40a6-a011-53bf09bab251.png)

Made only for himself as part of the study of the bitcoin protocol.
