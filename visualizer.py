#!/usr/bin/env python3

########################################################
# 0) http://127.0.0.1:5000/
########################################################

import json
from pathlib import Path

import geopandas as gpd
import pandas as pd
from flask import Flask, request, jsonify
from dash import Dash, dcc, html, Input, Output

import plotly.express as px
import plotly.graph_objects as go

########################################################
# 1) LOAD THE SWISS GEOJSON / GEODATAFRAME
########################################################
geojson_path = Path("maps/swissBOUNDARIES3D_1_3_TLM_KANTONSGEBIET.geojson")
if not geojson_path.exists():
    raise FileNotFoundError(f"GeoJSON not found: {geojson_path}")

gdf = gpd.read_file(geojson_path)

# Pre-compute centroid (lon, lat) for each KANTONSNUM in [1..26]
canton_centroids = {}
for kantonsnum in range(1, 27):
    subset = gdf[gdf["KANTONSNUM"] == kantonsnum]
    if subset.empty:
        continue
    c = subset.geometry.centroid.iloc[0]
    canton_centroids[kantonsnum] = (c.x, c.y)

swiss_geojson = json.loads(gdf.to_json())

########################################################
# 2) GLOBAL DATA STORE FOR GAME UPDATES
########################################################
app_data = {"latest_state": {}}

########################################################
# 2.5) DICTIONARY: REMAP GO ENGINE IDS => KANTONSNUM
########################################################
GO_TO_GEO = {
    25: 1,  3:  2,  11: 3,  21: 4,  18: 5,
    14: 6,  13: 7,  8:  8,  24: 9,  6:  10,
    17: 11, 4:  13, 5:  14, 16: 15, 1:  16,
    15: 17, 9:  18, 0:  19, 19: 20, 20: 21,
    22: 22, 23: 23, 12: 24, 7:  25, 10: 26,
}

# Debugging: Verify the mapping
print("GO_TO_GEO Mapping:", GO_TO_GEO)

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

dash_app.layout = html.Div([
    html.H3("Swiss MCTS Risk Visualization"),
    dcc.Graph(id="swiss-map"),
    dcc.Interval(id="interval-component", interval=2000, n_intervals=0)
], style={
    "backgroundColor": "#d3d3d3",  # Light gray background
    "height": "100vh",             # Full viewport height
    "padding": "20px"              # Padding for spacing
})
phase_map = {
    0: "Reinforcement",
    1: "Attack",
    2: "Maneuver",
    3: "End"
}
# Action constants
ATTACK_ACTION   = 1
MANEUVER_ACTION = 3

# Color lookup for phases
PHASE_COLORS = {
    "Reinforcement": "green",
    "Attack":        "orange",
    "Maneuver":      "purple",
    "End":           "gray",
}

@dash_app.callback(
    Output("swiss-map", "figure"),
    [Input("interval-component", "n_intervals")]
)
def update_map(_):
    """Called every 2s. Displays:
       - Choropleth (no legend)
       - Green troop labels
       - Arrows for Attack/Maneuver with circle markers
       - Top text with color-coded player/phase/troopsToPlace
    """
    state = app_data["latest_state"]
    if not state:
        # Return an empty choropleth map to prevent errors
        return px.choropleth_mapbox()

    ownership_list = state.get("Ownership", [])
    troop_counts   = state.get("TroopCounts", [])
    current_player = state.get("CurrentPlayer", 0)
    phase_index    = state.get("Phase", 0)
    
    # Track previous player and increment turn if player changes
    if "previous_player" not in app_data:
        app_data["previous_player"] = current_player
        app_data["turn_counter"] = 0

    if app_data["previous_player"] != current_player:
        app_data["turn_counter"] += 1
        app_data["previous_player"] = current_player

    turn_number = app_data["turn_counter"]

    troops_to_place = state.get("TroopsToPlace", 0)  # for Reinforcement

    phase_str   = phase_map.get(phase_index, f"Unknown({phase_index})")
    phase_color = PHASE_COLORS.get(phase_str, "black")

    # Build DF for [kantonsnum, owner, troops]
    rows = []
    for go_id, owner in enumerate(ownership_list):
        kantonsnum = GO_TO_GEO.get(go_id, 999)
        tc = troop_counts[go_id] if go_id < len(troop_counts) else 0
        rows.append({"kantonsnum": kantonsnum, "owner": owner, "troops": tc})
    df = pd.DataFrame(rows)

    # Hide "legend" by forcibly ignoring categories
    df["owner_str"] = df["owner"].astype(str)
    color_map = {"1": "red", "2": "blue", "0": "lightgray"}

    choropleth = px.choropleth_mapbox(
        df,
        geojson=swiss_geojson,
        locations="kantonsnum",
        color="owner_str",
        featureidkey="properties.KANTONSNUM",
        center={"lat": 46.8, "lon": 8.3},
        mapbox_style="carto-positron",
        zoom=6,
        hover_data=["troops"],
        color_discrete_map=color_map,
    )
    # Remove the built-in color scale & legend
    choropleth.update_layout(
        showlegend=False,
        coloraxis_showscale=False,
        margin={"r":0, "t":40, "l":0, "b":0},
    )

    # Convert to go.Figure
    fig = go.Figure(data=choropleth.data, layout=choropleth.layout)

    # (2) Draw an arrow for lastMove if Attack/Maneuver
    last_move = state.get("lastMove", {})
    a_type = last_move.get("ActionType", None)
    from_canton = last_move.get("FromCantonID", -1)
    to_canton   = last_move.get("ToCantonID", -1)

    arrow_col = None
    if a_type == ATTACK_ACTION:
        arrow_col = "orange"
    elif a_type == MANEUVER_ACTION:
        arrow_col = "purple"

    if arrow_col and from_canton >=0 and to_canton >=0:
        f_num = GO_TO_GEO.get(from_canton, 999)
        t_num = GO_TO_GEO.get(to_canton, 999)
        if f_num in canton_centroids and t_num in canton_centroids:
            fx, fy = canton_centroids[f_num]
            tx, ty = canton_centroids[t_num]
            # Debugging: Print coordinates
            print(f"Drawing arrow from KANTONSNUM {f_num} at ({fx}, {fy}) to {t_num} at ({tx}, {ty})")
            # Draw the line
            arrow_line = go.Scattermapbox(
                lon=[fx, tx],
                lat=[fy, ty],
                mode="lines",
                line=dict(color=arrow_col, width=3),
                hoverinfo="none"
            )
            fig.add_trace(arrow_line)
            
            # Add the arrowhead as a circle marker at the target canton
            arrow_head = go.Scattermapbox(
                lon=[tx],
                lat=[ty],
                mode="markers",
                marker=dict(
                    symbol="circle",  # Using circle as arrowhead
                    size=15,          # Adjust size as needed
                    color=arrow_col,  # Same color as the line
                    opacity=1         # Ensure full opacity
                ),
                hoverinfo="none"
            )
            fig.add_trace(arrow_head)

    # (1) Troop labels
    lat_vals = []
    lon_vals = []
    text_vals = []
    for _, row in df.iterrows():
        k_num = row["kantonsnum"]
        if k_num in canton_centroids:
            lon, lat = canton_centroids[k_num]
            text_vals.append(str(row["troops"]))
            lon_vals.append(lon)
            lat_vals.append(lat)
        else:
            lon_vals.append(None)
            lat_vals.append(None)
            text_vals.append("")

    # Add troop labels after arrows to render them on top
    scatter_troops = go.Scattermapbox(
        lat=lat_vals,
        lon=lon_vals,
        mode="text",
        text=text_vals,
        textfont=dict(color="green", size=12),
        hoverinfo="none"
    )
    fig.add_trace(scatter_troops)

    # Player color
    p_color = "red" if current_player == 1 else "blue"
    troops_text = ""
    if phase_str == "Reinforcement":
        troops_text = f"  (Troops: {troops_to_place})"

    # "Turn N - " in black
    fig.add_annotation(
        x=0.05, y=1.05, xref="paper", yref="paper",
        showarrow=False,
        text=f"Turn {turn_number} - ",
        font=dict(size=16, color="black", family="Arial")
    )
    # "Player=1" in player color
    fig.add_annotation(
        x=0.20, y=1.05, xref="paper", yref="paper",
        showarrow=False,
        text=f"Player={current_player}",
        font=dict(size=16, color=p_color, family="Arial")
    )
    # ", Phase=Attack" in phase color + optional troops if Reinforcement
    fig.add_annotation(
        x=0.45, y=1.05, xref="paper", yref="paper",
        showarrow=False,
        text=f", Phase={phase_str}{troops_text}",
        font=dict(size=16, color=phase_color, family="Arial")
    )

    return fig


########################################################
# 5) RUN THE APP
########################################################
if __name__ == "__main__":
    dash_app.run_server(host="0.0.0.0", port=5000, debug=True)
