-- AUTOGENERATED BY storx/dbx
-- DO NOT EDIT
CREATE TABLE nodes (
	id bytea NOT NULL,
	name text NOT NULL,
	public_address text NOT NULL,
	api_secret bytea NOT NULL,
	PRIMARY KEY ( id )
);

-- MAIN DATA --

-- NEW DATA --

INSERT INTO nodes (id, name, public_address, api_secret) VALUES (E'\\006\\223\\250R\\221\\005\\365\\377v>0\\266\\365\\216\\255?\\347\\244\\371?2\\264\\262\\230\\007<\\001\\262\\263\\237\\247n', 'node_name', '127.0.0.1:13000', E'\\153\\313\\233\\074\\327\\177\\136\\070\\346\\001');
