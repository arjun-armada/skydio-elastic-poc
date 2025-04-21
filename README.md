# skydio-elastic-poc


## Curl Request and Response for UI 

```
curl -X POST "http://localhost:9200/flight-media-join-flights/_search" -H "Content-Type: application/json" -d '
{
  "size": 0,
  "query": {
    "geo_bounding_box": {
      "location": {
        "top_left": {
          "lat": 49.3457868,
          "lon": -124.7844079
        },
        "bottom_right": {
          "lat": 24.396308,
          "lon": -66.93457
        }
      }
    }
  },
  "aggs": {
    "zoomed_points": {
      "geotile_grid": {
        "field": "location",
        "precision": 8
      },
      "aggs": {
        "sample_point": {
          "top_hits": {
            "_source": ["location", "FlightMedia.download_url"],
            "size": 1
          }
        }
      }
    }
  }
}'
```

# Response

```
{
    "took": 5,
    "timed_out": false,
    "_shards": {
        "total": 1,
        "successful": 1,
        "skipped": 0,
        "failed": 0
    },
    "hits": {
        "total": {
            "value": 298,
            "relation": "eq"
        },
        "max_score": null,
        "hits": []
    },
    "aggregations": {
        "zoomed_points": {
            "buckets": [
                {
                    "key": "8/40/98",
                    "doc_count": 137,
                    "sample_point": {
                        "hits": {
                            "total": {
                                "value": 137,
                                "relation": "eq"
                            },
                            "max_score": 1,
                            "hits": [
                                {
                                    "_index": "flight-media-join-flights",
                                    "_id": "6xYH8JUBUlHenIv9Hfib",
                                    "_score": 1,
                                    "_source": {
                                        "FlightMedia": {
                                            "download_url": "https://api.skydio.com/api/v0/media/download/88d7d06e-4e84-4e7a-bcc8-bdd45752a7b8"
                                        },
                                        "location": {
                                            "lat": 37.9896432,
                                            "lon": -122.5762647
                                        }
                                    }
                                }
                            ]
                        }
                    }
                },
                {
                    "key": "8/40/99",
                    "doc_count": 84,
                    "sample_point": {
                        "hits": {
                            "total": {
                                "value": 84,
                                "relation": "eq"
                            },
                            "max_score": 1,
                            "hits": [
                                {
                                    "_index": "flight-media-join-flights",
                                    "_id": "5RYH8JUBUlHenIv9HfiV",
                                    "_score": 1,
                                    "_source": {
                                        "FlightMedia": {
                                            "download_url": "https://api.skydio.com/api/v0/media/download/e17a68e7-3771-4faf-b8c3-3e80454339d4"
                                        },
                                        "location": {
                                            "lat": 37.4053061,
                                            "lon": -122.4235532
                                        }
                                    }
                                }
                            ]
                        }
                    }
                },
                {
                    "key": "8/41/89",
                    "doc_count": 42,
                    "sample_point": {
                        "hits": {
                            "total": {
                                "value": 42,
                                "relation": "eq"
                            },
                            "max_score": 1,
                            "hits": [
                                {
                                    "_index": "flight-media-join-flights",
                                    "_id": "4hYH8JUBUlHenIv9HfiD",
                                    "_score": 1,
                                    "_source": {
                                        "FlightMedia": {
                                            "download_url": "https://api.skydio.com/api/v0/media/download/e2dd56c5-f3e4-425a-953f-6f5931ddf6e0"
                                        },
                                        "location": {
                                            "lat": 47.5831066,
                                            "lon": -122.1557729
                                        }
                                    }
                                }
                            ]
                        }
                    }
                },
                {
                    "key": "8/41/99",
                    "doc_count": 35,
                    "sample_point": {
                        "hits": {
                            "total": {
                                "value": 35,
                                "relation": "eq"
                            },
                            "max_score": 1,
                            "hits": [
                                {
                                    "_index": "flight-media-join-flights",
                                    "_id": "PhYH8JUBUlHenIv9Hfn0",
                                    "_score": 1,
                                    "_source": {
                                        "FlightMedia": {
                                            "download_url": "https://api.skydio.com/api/v0/media/download/ffdcb356-42cd-47ad-b108-872846eeb8e9"
                                        },
                                        "location": {
                                            "lat": 36.9483817,
                                            "lon": -121.8762605
                                        }
                                    }
                                }
                            ]
                        }
                    }
                }
            ]
        }
    }
}

```
