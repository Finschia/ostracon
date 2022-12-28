# RPC spec

Please ensure you've first read the spec for [CometBFT RPC spec](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/rpc/README.md). Here only defines the difference between CometBFT.

## Support

Ostracon has only a Go implementation.

## Info Routes

### Block

Ostracon adds entropy to the [Block](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/rpc/README.md#block) response.

#### Parameters

- `height (integer)`: height of the requested block. If no height is specified the latest block will be used.

#### Request

##### HTTP

```sh
curl http://127.0.0.1:26657/block
curl http://127.0.0.1:26657/block?height=1
```

##### JSONRPC

```sh
curl -X POST https://localhost:26657 -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"block\",\"params\":{\"height\":\"1\"}}"
```

#### Response

```json
{
  "id": 0,
  "jsonrpc": "2.0",
  "result": {
    "block_id": {
      "hash": "112BC173FD838FB68EB43476816CD7B4C6661B6884A9E357B417EE957E1CF8F7",
      "parts": {
        "total": 1,
        "hash": "38D4B26B5B725C4F13571EFE022C030390E4C33C8CF6F88EDD142EA769642DBD"
      }
    },
    "block": {
      "header": {
        "version": {
          "block": "10",
          "app": "0"
        },
        "chain_id": "cosmoshub-2",
        "height": "12",
        "time": "2019-04-22T17:01:51.701356223Z",
        "last_block_id": {
          "hash": "112BC173FD838FB68EB43476816CD7B4C6661B6884A9E357B417EE957E1CF8F7",
          "parts": {
            "total": 1,
            "hash": "38D4B26B5B725C4F13571EFE022C030390E4C33C8CF6F88EDD142EA769642DBD"
          }
        },
        "last_commit_hash": "21B9BC845AD2CB2C4193CDD17BFC506F1EBE5A7402E84AD96E64171287A34812",
        "data_hash": "970886F99E77ED0D60DA8FCE0447C2676E59F2F77302B0C4AA10E1D02F18EF73",
        "validators_hash": "D658BFD100CA8025CFD3BECFE86194322731D387286FBD26E059115FD5F2BCA0",
        "next_validators_hash": "D658BFD100CA8025CFD3BECFE86194322731D387286FBD26E059115FD5F2BCA0",
        "consensus_hash": "0F2908883A105C793B74495EB7D6DF2EEA479ED7FC9349206A65CB0F9987A0B8",
        "app_hash": "223BF64D4A01074DC523A80E76B9BBC786C791FB0A1893AC5B14866356FCFD6C",
        "last_results_hash": "",
        "evidence_hash": "",
        "proposer_address": "D540AB022088612AC74B287D076DBFBC4A377A2E"
      },
      "data": [
        "yQHwYl3uCkKoo2GaChRnd+THLQ2RM87nEZrE19910Z28ABIUWW/t8AtIMwcyU0sT32RcMDI9GF0aEAoFdWF0b20SBzEwMDAwMDASEwoNCgV1YXRvbRIEMzEwMRCd8gEaagom61rphyEDoJPxlcjRoNDtZ9xMdvs+lRzFaHe2dl2P5R2yVCWrsHISQKkqX5H1zXAIJuC57yw0Yb03Fwy75VRip0ZBtLiYsUqkOsPUoQZAhDNP+6LY+RUwz/nVzedkF0S29NZ32QXdGv0="
      ],
      "evidence": [
        {
          "type": "string",
          "height": 0,
          "time": 0,
          "total_voting_power": 0,
          "validator": {
            "pub_key": {
              "type": "tendermint/PubKeyEd25519",
              "value": "A6DoBUypNtUAyEHWtQ9bFjfNg8Bo9CrnkUGl6k6OHN4="
            },
            "voting_power": 0,
            "address": "string"
          }
        }
      ],
      "last_commit": {
        "height": 0,
        "round": 0,
        "block_id": {
          "hash": "112BC173FD838FB68EB43476816CD7B4C6661B6884A9E357B417EE957E1CF8F7",
          "parts": {
            "total": 1,
            "hash": "38D4B26B5B725C4F13571EFE022C030390E4C33C8CF6F88EDD142EA769642DBD"
          }
        },
        "signatures": [
          {
            "type": 2,
            "height": "1262085",
            "round": 0,
            "block_id": {
              "hash": "112BC173FD838FB68EB43476816CD7B4C6661B6884A9E357B417EE957E1CF8F7",
              "parts": {
                "total": 1,
                "hash": "38D4B26B5B725C4F13571EFE022C030390E4C33C8CF6F88EDD142EA769642DBD"
              }
            },
            "timestamp": "2019-08-01T11:39:38.867269833Z",
            "validator_address": "000001E443FD237E4B616E2FA69DF4EE3D49A94F",
            "validator_index": 0,
            "signature": "DBchvucTzAUEJnGYpNvMdqLhBAHG4Px8BsOBB3J3mAFCLGeuG7uJqy+nVngKzZdPhPi8RhmE/xcw/M9DOJjEDg=="
          }
        ]
      },
      "entropy": {
        "round": 0,
        "proof": "03a9873d2c5fa9240cf987383b6127cc49cff64ec34f749aa71da80ced8584b9e43c866f96c07c83ba58e319572587485800190c79fcca8c0409e80b6af4279e4e9a889abcb4c1e2561499ad523bc4cbb2",
      }
    }
  }
}
```

### BlockByHash

Ostracon adds entropy to the [BlockByHash](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/rpc/README.md#blockbyhash) response.

#### Parameters

- `hash (string)`: Hash of the block to query for.

#### Request

##### HTTP

```sh
curl http://127.0.0.1:26657/block_by_hash?hash=0xD70952032620CC4E2737EB8AC379806359D8E0B17B0488F627997A0B043ABDED
```

##### JSONRPC

```sh
curl -X POST https://localhost:26657 -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"block_by_hash\",\"params\":{\"hash\":\"0xD70952032620CC4E2737EB8AC379806359D8E0B17B0488F627997A0B043ABDED\"}}"
```

#### Response

```json
{
  "id": 0,
  "jsonrpc": "2.0",
  "result": {
    "block_id": {
      "hash": "112BC173FD838FB68EB43476816CD7B4C6661B6884A9E357B417EE957E1CF8F7",
      "parts": {
        "total": 1,
        "hash": "38D4B26B5B725C4F13571EFE022C030390E4C33C8CF6F88EDD142EA769642DBD"
      }
    },
    "block": {
      "header": {
        "version": {
          "block": "10",
          "app": "0"
        },
        "chain_id": "cosmoshub-2",
        "height": "12",
        "time": "2019-04-22T17:01:51.701356223Z",
        "last_block_id": {
          "hash": "112BC173FD838FB68EB43476816CD7B4C6661B6884A9E357B417EE957E1CF8F7",
          "parts": {
            "total": 1,
            "hash": "38D4B26B5B725C4F13571EFE022C030390E4C33C8CF6F88EDD142EA769642DBD"
          }
        },
        "last_commit_hash": "21B9BC845AD2CB2C4193CDD17BFC506F1EBE5A7402E84AD96E64171287A34812",
        "data_hash": "970886F99E77ED0D60DA8FCE0447C2676E59F2F77302B0C4AA10E1D02F18EF73",
        "validators_hash": "D658BFD100CA8025CFD3BECFE86194322731D387286FBD26E059115FD5F2BCA0",
        "next_validators_hash": "D658BFD100CA8025CFD3BECFE86194322731D387286FBD26E059115FD5F2BCA0",
        "consensus_hash": "0F2908883A105C793B74495EB7D6DF2EEA479ED7FC9349206A65CB0F9987A0B8",
        "app_hash": "223BF64D4A01074DC523A80E76B9BBC786C791FB0A1893AC5B14866356FCFD6C",
        "last_results_hash": "",
        "evidence_hash": "",
        "proposer_address": "D540AB022088612AC74B287D076DBFBC4A377A2E"
      },
      "data": [
        "yQHwYl3uCkKoo2GaChRnd+THLQ2RM87nEZrE19910Z28ABIUWW/t8AtIMwcyU0sT32RcMDI9GF0aEAoFdWF0b20SBzEwMDAwMDASEwoNCgV1YXRvbRIEMzEwMRCd8gEaagom61rphyEDoJPxlcjRoNDtZ9xMdvs+lRzFaHe2dl2P5R2yVCWrsHISQKkqX5H1zXAIJuC57yw0Yb03Fwy75VRip0ZBtLiYsUqkOsPUoQZAhDNP+6LY+RUwz/nVzedkF0S29NZ32QXdGv0="
      ],
      "evidence": [
        {
          "type": "string",
          "height": 0,
          "time": 0,
          "total_voting_power": 0,
          "validator": {
            "pub_key": {
              "type": "tendermint/PubKeyEd25519",
              "value": "A6DoBUypNtUAyEHWtQ9bFjfNg8Bo9CrnkUGl6k6OHN4="
            },
            "voting_power": 0,
            "address": "string"
          }
        }
      ],
      "last_commit": {
        "height": 0,
        "round": 0,
        "block_id": {
          "hash": "112BC173FD838FB68EB43476816CD7B4C6661B6884A9E357B417EE957E1CF8F7",
          "parts": {
            "total": 1,
            "hash": "38D4B26B5B725C4F13571EFE022C030390E4C33C8CF6F88EDD142EA769642DBD"
          }
        },
        "signatures": [
          {
            "type": 2,
            "height": "1262085",
            "round": 0,
            "block_id": {
              "hash": "112BC173FD838FB68EB43476816CD7B4C6661B6884A9E357B417EE957E1CF8F7",
              "parts": {
                "total": 1,
                "hash": "38D4B26B5B725C4F13571EFE022C030390E4C33C8CF6F88EDD142EA769642DBD"
              }
            },
            "timestamp": "2019-08-01T11:39:38.867269833Z",
            "validator_address": "000001E443FD237E4B616E2FA69DF4EE3D49A94F",
            "validator_index": 0,
            "signature": "DBchvucTzAUEJnGYpNvMdqLhBAHG4Px8BsOBB3J3mAFCLGeuG7uJqy+nVngKzZdPhPi8RhmE/xcw/M9DOJjEDg=="
          }
        ]
      },
      "entropy": {
        "round": 0,
        "proof": "03a9873d2c5fa9240cf987383b6127cc49cff64ec34f749aa71da80ced8584b9e43c866f96c07c83ba58e319572587485800190c79fcca8c0409e80b6af4279e4e9a889abcb4c1e2561499ad523bc4cbb2",
      }
    }
  }
}
```