# BRPC

An RPC protocol over TCP with support for bit-level control.

## Installation

`go get github.com/josephprichard/brpc`

## Usage

Define a data spec using the 'message' structure.

```
message Board struct {
    required turn b1 = 1; // a 1 bit field, it can either be white or black's turn
    required color [b64]2 = 2; // a 128 bit field that will be deserialized as a 64 length array
}

message Move struct {
    required row b3 = 1; // a 3 bit field, since this can only contain values 0-7
    required col b3 = 2;
    required disc b1 = 3; // a 1 bit field, this can either be white or black
}

message GameResult struct {
    required winnerId b128 = 1;
    required loserId b128 = 2;
    required winerElo float32 = 3; // a float is a 32 or 64 bit float
    required loserElo float32 = 4;
    required isDraw int1 = 5;
}

message Game struct {
    required id b128 = 4; // we can add fields in any order in the program, the assigned 'ord' value determines the order in serialization
    required whiteId b128 = 1; // storing a uuid in a 128 bit field
    required blackId b128 = 2;
    required board OthelloBoard = 3; // compose a larger message from smaller messages, this is stored like we just copied the fields in here
    required moves []Move = 5; // a variable length array, the size will be packed into 4 bytes at the start of the array
    optional GameResult = 6; // a field can be optional, in which case it has 1 bit packed in front of it to represent if the data is present or not
}
```

Define an RPC service using the 'service' structure.
```
// a serivice that performs operations for othello games over the wire
// notice that all arguments and return values are performed through named tuples, this can be thought of as an anonymous message defined for each operation
service OthelloService {
    MakeMove (move Move) -> (err b8); 
    GetGame (id b128) -> (err b8 = 1, game Game = 2); // 'ord' is required when more than one tuple field is provided, otherwise it defaults to 1
}
```