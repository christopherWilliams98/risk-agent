#!/usr/bin/env python3

import json
from pathlib import Path

import geopandas as gpd
import pandas as pd
from flask import Flask, request, jsonify
from dash import Dash, dcc, html, Input, Output
import plotly.express as px

########################################################
# 1) LOAD THE SWISS GEOJSON / GEODATAFRAME
########################################################
geojson_path = Path("maps/swissBOUNDARIES3D_1_3_TLM_KANTONSGEBIET.geojson")
if not geojson_path.exists():
    raise FileNotFoundError(f"GeoJSON not found: {geojson_path}")

gdf = gpd.read_file(geojson_path)
swiss_geojson = json.loads(gdf.to_json())  # Convert to geojson dict for Plotly

########################################################
# 2) GLOBAL DATA STORE FOR GAME UPDATES
########################################################
app_data = {
    "latest_state": {}
}

########################################################
# 2.5) DICTIONARY: REMAP GO ENGINE IDS => KANTONSNUM
########################################################
# Fill in carefully so that each Go ID matches the KANTONSNUM in your GeoJSON.
# The lines below are just examples. You MUST update them to reflect your actual
# mapping from the Go code. The user-supplied snippet is partial, so fill them out
# for all 26 Cantons.
GO_TO_GEO = {
    # e.g. Go ID=25 => Zürich => KANTONSNUM=1
    25: 1,
    # e.g. Go ID=3  => Bern   => KANTONSNUM=2
    3:  2,
    # e.g. Go ID=11 => Luzern => KANTONSNUM=3
    11: 3,
    # e.g. Go ID=21 => Uri    => KANTONSNUM=4
    21: 4,
    # e.g. Go ID=18 => Schwyz => KANTONSNUM=5
    18: 5,
    # e.g. Go ID=14 => Obwalden => KANTONSNUM=6
    14: 6,
    # e.g. Go ID=13 => Nidwalden => KANTONSNUM=7
    13: 7,
    # e.g. Go ID=8  => Glarus => KANTONSNUM=8
    8:  8,
    # e.g. Go ID=24 => Zug    => KANTONSNUM=9
    24: 9,
    # e.g. Go ID=6  => Fribourg => KANTONSNUM=10
    6:  10,
    # e.g. Go ID=17 => Solothurn => KANTONSNUM=11
    17: 11,
    # Fill out the rest...
    # e.g. Go ID=23 => Valais => KANTONSNUM=23
    23: 23,
    # Go ID=0 => Aargau => maybe KANTONSNUM=19 if your JSON shows that
    0: 19,
    # etc. For anything unknown, you can default to 999 or skip
}

########################################################
# 3) FLASK: /visualhook
########################################################
server = Flask(__name__)

@server.route("/visualhook", methods=["POST"])
def visualhook():
    data = request.get_json()
    if not data:
        return jsonify({"error": "No JSON data received"}), 400

    app_data["latest_state"] = data
    return jsonify({"ok": True})

########################################################
# 4) DASH APP
########################################################
dash_app = Dash(__name__, server=server, url_base_pathname="/")

# We'll provide a small legend for player colors, 
# a Graph for the map, and an Interval to poll for updates.
dash_app.layout = html.Div([
    html.H3("Swiss Risk Visualization"),
    html.Div("Player1: Red, Player2: Blue", style={"fontWeight":"bold"}),
    dcc.Graph(id="swiss-map"),
    dcc.Interval(id="interval-component", interval=2000, n_intervals=0)
])

# Phase index → English label
phase_map = {
    0: "Reinforcement",
    1: "Attack",
    2: "Maneuver",
    3: "End"
}

@dash_app.callback(
    Output("swiss-map", "figure"),
    [Input("interval-component", "n_intervals")]
)
def update_map(_):
    """Called every 2 seconds. Builds a Swiss map with discrete color for owners."""
    state = app_data["latest_state"]
    if not state:
        # No data yet => empty figure
        return px.choropleth_mapbox()

    ownership_list = state.get("Ownership", [])
    troop_counts   = state.get("TroopCounts", [])
    current_player = state.get("CurrentPlayer", 0)
    phase_index    = state.get("Phase", 0)
    # Optionally, if your Go code sets state["Turn"], read it:
    turn_number    = state.get("Turn", 0)

    # Translate numeric phase → English word
    phase_str = phase_map.get(phase_index, f"Unknown({phase_index})")

    # Build a DataFrame linking each Go ID to the correct KANTONSNUM in the geojson
    rows = []
    for go_id, owner in enumerate(ownership_list):
        kantonsnum = GO_TO_GEO.get(go_id, 999)  # default 999 if missing
        troops = troop_counts[go_id] if go_id < len(troop_counts) else 0
        rows.append({
            "kantonsnum": kantonsnum,  # Key for the shape
            "owner":      owner,
            "troops":     troops
        })

    df = pd.DataFrame(rows)

    # We'll map integer owners to discrete colors. 
    df["owner_str"] = df["owner"].astype(str)

    # Example discrete color map:
    color_map = {
        "1": "red",
        "2": "blue",
        "0": "lightgray"  # For unowned or "player 0"
        # Add "3": "green", etc. if you have more players
    }

    # Build a discrete choropleth
    fig = px.choropleth_mapbox(
        df,
        geojson=swiss_geojson,
        locations="kantonsnum",             # column in df
        color="owner_str",
        featureidkey="properties.KANTONSNUM",  # must match the property in the GeoJSON
        center={"lat": 46.8, "lon": 8.3},
        mapbox_style="carto-positron",
        zoom=6,
        hover_data=["troops"],
        color_discrete_map=color_map
    )

    fig.update_layout(
        title=f"Turn {turn_number} - Player={current_player}, Phase={phase_str}",
        margin={"r":0,"t":40,"l":0,"b":0},
        coloraxis_showscale=False
    )

    return fig

########################################################
# 5) RUN THE APP
########################################################
if __name__ == "__main__":
    dash_app.run_server(host="0.0.0.0", port=5000, debug=True)
