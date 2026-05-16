-- Remove only seeded SKUs; keep any user-created records.
DELETE FROM materials WHERE sku IN (
    'PLT-AH36-1020', 'PLT-AH36-1220', 'PLT-DH36-1620',
    'WLD-E7018-350', 'WLD-E7018-400',
    'PNT-EPX-MAR20', 'PNT-AFO-RED20',
    'PPE-PIP-SCH40', 'BLT-HEX-M20',
    'GAS-ACT-50L', 'GAS-OXY-50L',
    'PLT-SS316-0810', 'CBL-PWR-35MM',
    'INS-RCK-50MM', 'VLV-GTR-DN50'
);
