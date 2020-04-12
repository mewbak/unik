// SPDX-License-Identifier: Unlicense OR MIT

package main

var shaders = map[[2]string][2]string{
	{"precision mediump float;\nprecision highp int;\n\nstruct Color\n{\n    vec4 _color;\n};\n\nuniform Color _12;\n\nvarying vec2 vUV;\n\nvoid main()\n{\n    gl_FragData[0] = _12._color;\n}\n\n", "\nstruct Block\n{\n    vec4 transform;\n    vec4 uvTransform;\n    float z;\n};\n\nuniform Block _24;\n\nattribute vec2 pos;\nvarying vec2 vUV;\nattribute vec2 uv;\n\nvec4 toClipSpace(vec4 pos_1)\n{\n    return pos_1;\n}\n\nvoid main()\n{\n    vec2 p = (pos * _24.transform.xy) + _24.transform.zw;\n    vec4 param = vec4(p, _24.z, 1.0);\n    gl_Position = toClipSpace(param);\n    vUV = (uv * _24.uvTransform.xy) + _24.uvTransform.zw;\n}\n\n"}: {`FRAG
DCL OUT[0], COLOR
DCL CONST[1][0]
IMM[0] UINT32 {0, 0, 0, 0}
  0: MOV OUT[0], CONST[1][0]
  1: END`, `VERT
DCL IN[0]
DCL OUT[0], POSITION
DCL CONST[1][0..2]
DCL TEMP[0], LOCAL
IMM[0] FLT32 {    1.0000,     0.0000,     0.0000,     0.0000}
IMM[1] UINT32 {0, 32, 0, 0}
  0: MOV TEMP[0].w, IMM[0].xxxx
  1: MAD TEMP[0].xy, IN[0].xyyy, CONST[1][0].xyyy, CONST[1][0].zwww
  2: MOV TEMP[0].z, CONST[1][2].xxxx
  3: MOV OUT[0], TEMP[0]
  4: END`},
	{"precision mediump float;\nprecision highp int;\n\nuniform mediump sampler2D tex;\n\nvarying vec2 vUV;\n\nvoid main()\n{\n    gl_FragData[0] = texture2D(tex, vUV);\n}\n\n", "\nstruct Block\n{\n    vec4 transform;\n    vec4 uvTransform;\n    float z;\n};\n\nuniform Block _24;\n\nattribute vec2 pos;\nvarying vec2 vUV;\nattribute vec2 uv;\n\nvec4 toClipSpace(vec4 pos_1)\n{\n    return pos_1;\n}\n\nvoid main()\n{\n    vec2 p = (pos * _24.transform.xy) + _24.transform.zw;\n    vec4 param = vec4(p, _24.z, 1.0);\n    gl_Position = toClipSpace(param);\n    vUV = (uv * _24.uvTransform.xy) + _24.uvTransform.zw;\n}\n\n"}: {`FRAG
DCL IN[0].xy, GENERIC[9], PERSPECTIVE
DCL OUT[0], COLOR
DCL SAMP[0]
DCL SVIEW[0], 2D, FLOAT
DCL TEMP[0], LOCAL
  0: MOV TEMP[0].xy, IN[0].xyyy
  1: TEX TEMP[0], TEMP[0], SAMP[0], 2D
  2: MOV OUT[0], TEMP[0]
  3: END`, `VERT
DCL IN[0]
DCL IN[1]
DCL OUT[0], POSITION
DCL OUT[1].xy, GENERIC[9]
DCL CONST[1][0..2]
DCL TEMP[0..1], LOCAL
IMM[0] FLT32 {    1.0000,     0.0000,     0.0000,     0.0000}
IMM[1] UINT32 {0, 32, 16, 0}
  0: MOV TEMP[0].w, IMM[0].xxxx
  1: MAD TEMP[0].xy, IN[0].xyyy, CONST[1][0].xyyy, CONST[1][0].zwww
  2: MOV TEMP[0].z, CONST[1][2].xxxx
  3: MAD TEMP[1].xy, IN[1].xyyy, CONST[1][1].xyyy, CONST[1][1].zwww
  4: MOV OUT[0], TEMP[0]
  5: MOV OUT[1].xy, TEMP[1].xyxx
  6: END`},
	{"precision mediump float;\nprecision highp int;\n\nstruct Color\n{\n    vec4 _color;\n};\n\nuniform Color _12;\n\nuniform mediump sampler2D cover;\n\nvarying highp vec2 vCoverUV;\nvarying vec2 vUV;\n\nvoid main()\n{\n    gl_FragData[0] = _12._color;\n    float cover_1 = abs(texture2D(cover, vCoverUV).x);\n    gl_FragData[0] *= cover_1;\n}\n\n", "\nstruct Block\n{\n    vec4 transform;\n    vec4 uvCoverTransform;\n    vec4 uvTransform;\n    float z;\n};\n\nuniform Block _66;\n\nattribute vec2 pos;\nvarying vec2 vUV;\nattribute vec2 uv;\nvarying vec2 vCoverUV;\n\nvec4 toClipSpace(vec4 pos_1)\n{\n    return pos_1;\n}\n\nvec3[2] fboTextureTransform()\n{\n    vec3 t[2];\n    t[0] = vec3(1.0, 0.0, 0.0);\n    t[1] = vec3(0.0, 1.0, 0.0);\n    return t;\n}\n\nvec3 transform3x2(vec3 t[2], vec3 v)\n{\n    return vec3(dot(t[0], v), dot(t[1], v), dot(vec3(0.0, 0.0, 1.0), v));\n}\n\nvoid main()\n{\n    vec4 param = vec4((pos * _66.transform.xy) + _66.transform.zw, _66.z, 1.0);\n    gl_Position = toClipSpace(param);\n    vUV = (uv * _66.uvTransform.xy) + _66.uvTransform.zw;\n    vec3 fboTrans[2] = fboTextureTransform();\n    vec3 param_1[2] = fboTrans;\n    vec3 param_2 = vec3(uv, 1.0);\n    vec3 uv3 = transform3x2(param_1, param_2);\n    vCoverUV = ((uv3 * vec3(_66.uvCoverTransform.xy, 1.0)) + vec3(_66.uvCoverTransform.zw, 0.0)).xy;\n}\n\n"}: {`FRAG
DCL IN[0].xy, GENERIC[9], PERSPECTIVE
DCL OUT[0], COLOR
DCL SAMP[0]
DCL SVIEW[0], 2D, FLOAT
DCL CONST[1][0]
DCL TEMP[0], LOCAL
IMM[0] UINT32 {0, 0, 0, 0}
  0: MOV TEMP[0].xy, IN[0].xyyy
  1: TEX TEMP[0].x, TEMP[0], SAMP[0], 2D
  2: MOV TEMP[0].x, |TEMP[0].xxxx|
  3: MUL TEMP[0], CONST[1][0], TEMP[0].xxxx
  4: MOV OUT[0], TEMP[0]
  5: END`, `VERT
DCL IN[0]
DCL IN[1]
DCL OUT[0], POSITION
DCL OUT[1].xy, GENERIC[9]
DCL CONST[1][0..3]
DCL TEMP[0..3], LOCAL
IMM[0] FLT32 {    1.0000,     0.0000,     0.0000,     0.0000}
IMM[1] UINT32 {0, 48, 16, 0}
  0: MOV TEMP[0].w, IMM[0].xxxx
  1: MAD TEMP[0].xy, IN[0].xyyy, CONST[1][0].xyyy, CONST[1][0].zwww
  2: MOV TEMP[0].z, CONST[1][3].xxxx
  3: MOV TEMP[1].z, IMM[0].xxxx
  4: MOV TEMP[1].xy, IN[1].xyxx
  5: DP3 TEMP[2].x, IMM[0].xyyy, TEMP[1].xyzz
  6: DP3 TEMP[3].x, IMM[0].yxyy, TEMP[1].xyzz
  7: MOV TEMP[2].y, TEMP[3].xxxx
  8: DP3 TEMP[1].x, IMM[0].yyxx, TEMP[1].xyzz
  9: MOV TEMP[2].z, TEMP[1].xxxx
 10: MOV TEMP[1].z, IMM[0].xxxx
 11: MOV TEMP[1].xy, CONST[1][1].xyxx
 12: MOV TEMP[3].z, IMM[0].yyyy
 13: MOV TEMP[3].xy, CONST[1][1].zwzz
 14: MAD TEMP[1].xy, TEMP[2].xyzz, TEMP[1].xyzz, TEMP[3].xyzz
 15: MOV OUT[1].xy, TEMP[1].xyxx
 16: MOV OUT[0], TEMP[0]
 17: END`},
	{"precision mediump float;\nprecision highp int;\n\nuniform mediump sampler2D tex;\nuniform mediump sampler2D cover;\n\nvarying vec2 vUV;\nvarying highp vec2 vCoverUV;\n\nvoid main()\n{\n    gl_FragData[0] = texture2D(tex, vUV);\n    float cover_1 = abs(texture2D(cover, vCoverUV).x);\n    gl_FragData[0] *= cover_1;\n}\n\n", "\nstruct Block\n{\n    vec4 transform;\n    vec4 uvCoverTransform;\n    vec4 uvTransform;\n    float z;\n};\n\nuniform Block _66;\n\nattribute vec2 pos;\nvarying vec2 vUV;\nattribute vec2 uv;\nvarying vec2 vCoverUV;\n\nvec4 toClipSpace(vec4 pos_1)\n{\n    return pos_1;\n}\n\nvec3[2] fboTextureTransform()\n{\n    vec3 t[2];\n    t[0] = vec3(1.0, 0.0, 0.0);\n    t[1] = vec3(0.0, 1.0, 0.0);\n    return t;\n}\n\nvec3 transform3x2(vec3 t[2], vec3 v)\n{\n    return vec3(dot(t[0], v), dot(t[1], v), dot(vec3(0.0, 0.0, 1.0), v));\n}\n\nvoid main()\n{\n    vec4 param = vec4((pos * _66.transform.xy) + _66.transform.zw, _66.z, 1.0);\n    gl_Position = toClipSpace(param);\n    vUV = (uv * _66.uvTransform.xy) + _66.uvTransform.zw;\n    vec3 fboTrans[2] = fboTextureTransform();\n    vec3 param_1[2] = fboTrans;\n    vec3 param_2 = vec3(uv, 1.0);\n    vec3 uv3 = transform3x2(param_1, param_2);\n    vCoverUV = ((uv3 * vec3(_66.uvCoverTransform.xy, 1.0)) + vec3(_66.uvCoverTransform.zw, 0.0)).xy;\n}\n\n"}: {`FRAG
DCL IN[0], GENERIC[9], PERSPECTIVE
DCL OUT[0], COLOR
DCL SAMP[0]
DCL SAMP[1]
DCL SVIEW[0], 2D, FLOAT
DCL SVIEW[1], 2D, FLOAT
DCL TEMP[0..1], LOCAL
  0: MOV TEMP[0].xy, IN[0].zwww
  1: TEX TEMP[0], TEMP[0], SAMP[0], 2D
  2: MOV TEMP[1].xy, IN[0].xyyy
  3: TEX TEMP[1].x, TEMP[1], SAMP[1], 2D
  4: MOV TEMP[1].x, |TEMP[1].xxxx|
  5: MUL TEMP[0], TEMP[0], TEMP[1].xxxx
  6: MOV OUT[0], TEMP[0]
  7: END`, `VERT
DCL IN[0]
DCL IN[1]
DCL OUT[0], POSITION
DCL OUT[1], GENERIC[9]
DCL CONST[1][0..3]
DCL TEMP[0..4], LOCAL
IMM[0] FLT32 {    1.0000,     0.0000,     0.0000,     0.0000}
IMM[1] UINT32 {0, 48, 32, 16}
  0: MOV TEMP[0].w, IMM[0].xxxx
  1: MAD TEMP[0].xy, IN[0].xyyy, CONST[1][0].xyyy, CONST[1][0].zwww
  2: MOV TEMP[0].z, CONST[1][3].xxxx
  3: MAD TEMP[1].xy, IN[1].xyyy, CONST[1][2].xyyy, CONST[1][2].zwww
  4: MOV TEMP[2].z, IMM[0].xxxx
  5: MOV TEMP[2].xy, IN[1].xyxx
  6: DP3 TEMP[3].x, IMM[0].xyyy, TEMP[2].xyzz
  7: DP3 TEMP[4].x, IMM[0].yxyy, TEMP[2].xyzz
  8: MOV TEMP[3].y, TEMP[4].xxxx
  9: DP3 TEMP[2].x, IMM[0].yyxx, TEMP[2].xyzz
 10: MOV TEMP[3].z, TEMP[2].xxxx
 11: MOV TEMP[2].z, IMM[0].xxxx
 12: MOV TEMP[2].xy, CONST[1][1].xyxx
 13: MOV TEMP[4].z, IMM[0].yyyy
 14: MOV TEMP[4].xy, CONST[1][1].zwzz
 15: MAD TEMP[2].xy, TEMP[3].xyzz, TEMP[2].xyzz, TEMP[4].xyzz
 16: MOV OUT[1].xy, TEMP[2].xyxx
 17: MOV OUT[0], TEMP[0]
 18: MOV OUT[1].zw, TEMP[1].xxxy
 19: END`},
	{"precision mediump float;\nprecision highp int;\n\nuniform mediump sampler2D cover;\n\nvarying highp vec2 vUV;\n\nvoid main()\n{\n    float cover_1 = abs(texture2D(cover, vUV).x);\n    gl_FragData[0].x = cover_1;\n}\n\n", "\nstruct Block\n{\n    vec4 uvTransform;\n    vec4 subUVTransform;\n};\n\nuniform Block _101;\n\nattribute vec2 pos;\nattribute vec2 uv;\nvarying vec2 vUV;\n\nvec3[2] fboTransform()\n{\n    vec3 t[2];\n    t[0] = vec3(1.0, 0.0, 0.0);\n    t[1] = vec3(0.0, -1.0, 0.0);\n    return t;\n}\n\nvec3 transform3x2(vec3 t[2], vec3 v)\n{\n    return vec3(dot(t[0], v), dot(t[1], v), dot(vec3(0.0, 0.0, 1.0), v));\n}\n\nvec3[2] fboTextureTransform()\n{\n    vec3 t[2];\n    t[0] = vec3(1.0, 0.0, 0.0);\n    t[1] = vec3(0.0, 1.0, 0.0);\n    return t;\n}\n\nvoid main()\n{\n    vec3 fboTrans[2] = fboTransform();\n    vec3 param[2] = fboTrans;\n    vec3 param_1 = vec3(pos, 1.0);\n    vec3 p = transform3x2(param, param_1);\n    gl_Position = vec4(p, 1.0);\n    vec3 fboTexTrans[2] = fboTextureTransform();\n    vec3 param_2[2] = fboTexTrans;\n    vec3 param_3 = vec3(uv, 1.0);\n    vec3 uv3 = transform3x2(param_2, param_3);\n    vUV = (uv3.xy * _101.subUVTransform.xy) + _101.subUVTransform.zw;\n    vec3 param_4[2] = fboTexTrans;\n    vec3 param_5 = vec3(vUV, 1.0);\n    vUV = transform3x2(param_4, param_5).xy;\n    vUV = (vUV * _101.uvTransform.xy) + _101.uvTransform.zw;\n}\n\n"}: {`FRAG
DCL IN[0].xy, GENERIC[9], PERSPECTIVE
DCL OUT[0], COLOR
DCL SAMP[0]
DCL SVIEW[0], 2D, FLOAT
DCL TEMP[0], LOCAL
  0: MOV TEMP[0].xy, IN[0].xyyy
  1: TEX TEMP[0].x, TEMP[0], SAMP[0], 2D
  2: MOV TEMP[0].x, |TEMP[0].xxxx|
  3: MOV OUT[0], TEMP[0]
  4: END`, `VERT
DCL IN[0]
DCL IN[1]
DCL OUT[0], POSITION
DCL OUT[1].xy, GENERIC[9]
DCL CONST[1][0..1]
DCL TEMP[0..3], LOCAL
IMM[0] FLT32 {    1.0000,     0.0000,    -1.0000,     0.0000}
IMM[1] UINT32 {0, 16, 0, 0}
  0: MOV TEMP[0].z, IMM[0].xxxx
  1: MOV TEMP[0].xy, IN[0].xyxx
  2: DP3 TEMP[1].x, IMM[0].xyyy, TEMP[0].xyzz
  3: DP3 TEMP[2].x, IMM[0].yzyy, TEMP[0].xyzz
  4: MOV TEMP[1].y, TEMP[2].xxxx
  5: DP3 TEMP[0].x, IMM[0].yyxx, TEMP[0].xyzz
  6: MOV TEMP[1].z, TEMP[0].xxxx
  7: MOV TEMP[0].w, IMM[0].xxxx
  8: MOV TEMP[0].xyz, TEMP[1].xyzx
  9: MOV TEMP[1].z, IMM[0].xxxx
 10: MOV TEMP[1].xy, IN[1].xyxx
 11: DP3 TEMP[2].x, IMM[0].xyyy, TEMP[1].xyzz
 12: DP3 TEMP[1].x, IMM[0].yxyy, TEMP[1].xyzz
 13: MOV TEMP[2].y, TEMP[1].xxxx
 14: MAD TEMP[1].xy, TEMP[2].xyyy, CONST[1][1].xyyy, CONST[1][1].zwww
 15: MOV TEMP[2].z, IMM[0].xxxx
 16: MOV TEMP[2].xy, TEMP[1].xyxx
 17: DP3 TEMP[3].x, IMM[0].xyyy, TEMP[2].xyzz
 18: DP3 TEMP[2].x, IMM[0].yxyy, TEMP[2].xyzz
 19: MOV TEMP[3].y, TEMP[2].xxxx
 20: MAD TEMP[1].xy, TEMP[3].xyyy, CONST[1][0].xyyy, CONST[1][0].zwww
 21: MOV OUT[0], TEMP[0]
 22: MOV OUT[1].xy, TEMP[1].xyxx
 23: END`},
	{"precision mediump float;\nprecision highp int;\n\nvarying vec2 vTo;\nvarying vec2 vFrom;\nvarying vec2 vCtrl;\n\nvoid main()\n{\n    float dx = vTo.x - vFrom.x;\n    bool increasing = vTo.x >= vFrom.x;\n    bvec2 _35 = bvec2(increasing);\n    vec2 left = vec2(_35.x ? vFrom.x : vTo.x, _35.y ? vFrom.y : vTo.y);\n    bvec2 _41 = bvec2(increasing);\n    vec2 right = vec2(_41.x ? vTo.x : vFrom.x, _41.y ? vTo.y : vFrom.y);\n    vec2 extent = clamp(vec2(vFrom.x, vTo.x), vec2(-0.5), vec2(0.5));\n    float midx = mix(extent.x, extent.y, 0.5);\n    float x0 = midx - left.x;\n    vec2 p1 = vCtrl - left;\n    vec2 v = right - vCtrl;\n    float t = x0 / (p1.x + sqrt((p1.x * p1.x) + ((v.x - p1.x) * x0)));\n    float y = mix(mix(left.y, vCtrl.y, t), mix(vCtrl.y, right.y, t), t);\n    vec2 d_half = mix(p1, v, vec2(t));\n    float dy = d_half.y / d_half.x;\n    float width = extent.y - extent.x;\n    dy = abs(dy * width);\n    vec4 sides = vec4((dy * 0.5) + y, (dy * (-0.5)) + y, (0.5 - y) / dy, ((-0.5) - y) / dy);\n    sides = clamp(sides + vec4(0.5), vec4(0.0), vec4(1.0));\n    float area = 0.5 * ((((sides.z - (sides.z * sides.y)) + 1.0) - sides.x) + (sides.x * sides.w));\n    area *= width;\n    if (width == 0.0)\n    {\n        area = 0.0;\n    }\n    gl_FragData[0].x = area;\n}\n\n", "\nstruct Block\n{\n    vec4 transform;\n    vec2 pathOffset;\n};\n\nuniform Block _16;\n\nattribute vec2 from;\nattribute vec2 ctrl;\nattribute vec2 to;\nattribute float maxy;\nattribute float corner;\nvarying vec2 vFrom;\nvarying vec2 vCtrl;\nvarying vec2 vTo;\n\nvoid main()\n{\n    vec2 from_1 = from + _16.pathOffset;\n    vec2 ctrl_1 = ctrl + _16.pathOffset;\n    vec2 to_1 = to + _16.pathOffset;\n    float maxy_1 = maxy + _16.pathOffset.y;\n    float c = corner;\n    vec2 pos;\n    if (c >= 0.375)\n    {\n        c -= 0.5;\n        pos.y = maxy_1 + 1.0;\n    }\n    else\n    {\n        pos.y = min(min(from_1.y, ctrl_1.y), to_1.y) - 1.0;\n    }\n    if (c >= 0.125)\n    {\n        pos.x = max(max(from_1.x, ctrl_1.x), to_1.x) + 1.0;\n    }\n    else\n    {\n        pos.x = min(min(from_1.x, ctrl_1.x), to_1.x) - 1.0;\n    }\n    vFrom = from_1 - pos;\n    vCtrl = ctrl_1 - pos;\n    vTo = to_1 - pos;\n    pos = (pos * _16.transform.xy) + _16.transform.zw;\n    gl_Position = vec4(pos, 1.0, 1.0);\n}\n\n"}: {`FRAG
DCL IN[0], GENERIC[9], PERSPECTIVE
DCL IN[1].xy, GENERIC[10], PERSPECTIVE
DCL OUT[0], COLOR
DCL TEMP[0..7], LOCAL
IMM[0] FLT32 {   -0.5000,     0.5000,     1.0000,     0.0000}
  0: FSGE TEMP[0].x, IN[1].xxxx, IN[0].zzzz
  1: UIF TEMP[0].xxxx
  2:   MOV TEMP[1].x, IN[0].zzzz
  3: ELSE
  4:   MOV TEMP[1].x, IN[1].xxxx
  5: ENDIF
  6: UIF TEMP[0].xxxx
  7:   MOV TEMP[2].x, IN[0].wwww
  8: ELSE
  9:   MOV TEMP[2].x, IN[1].yyyy
 10: ENDIF
 11: MOV TEMP[3].x, TEMP[1].xxxx
 12: MOV TEMP[3].y, TEMP[2].xxxx
 13: UIF TEMP[0].xxxx
 14:   MOV TEMP[4].x, IN[1].xxxx
 15: ELSE
 16:   MOV TEMP[4].x, IN[0].zzzz
 17: ENDIF
 18: UIF TEMP[0].xxxx
 19:   MOV TEMP[0].x, IN[1].yyyy
 20: ELSE
 21:   MOV TEMP[0].x, IN[0].wwww
 22: ENDIF
 23: MOV TEMP[4].x, TEMP[4].xxxx
 24: MOV TEMP[4].y, TEMP[0].xxxx
 25: MOV TEMP[5].x, IN[0].zzzz
 26: MOV TEMP[5].y, IN[1].xxxx
 27: MAX TEMP[5].xy, TEMP[5].xyyy, IMM[0].xxxx
 28: MIN TEMP[5].xy, TEMP[5].xyyy, IMM[0].yyyy
 29: LRP TEMP[6].x, IMM[0].yyyy, TEMP[5].yyyy, TEMP[5].xxxx
 30: ADD TEMP[1].x, TEMP[6].xxxx, -TEMP[1].xxxx
 31: ADD TEMP[3].xy, IN[0].xyyy, -TEMP[3].xyyy
 32: ADD TEMP[4].xy, TEMP[4].xyyy, -IN[0].xyyy
 33: ADD TEMP[6].x, TEMP[4].xxxx, -TEMP[3].xxxx
 34: MUL TEMP[7].x, TEMP[3].xxxx, TEMP[3].xxxx
 35: MAD TEMP[6].x, TEMP[6].xxxx, TEMP[1].xxxx, TEMP[7].xxxx
 36: RSQ TEMP[6].x, |TEMP[6].xxxx|
 37: RCP TEMP[6].x, TEMP[6].xxxx
 38: ADD TEMP[6].x, TEMP[3].xxxx, TEMP[6].xxxx
 39: RCP TEMP[6].x, TEMP[6].xxxx
 40: MUL TEMP[1].x, TEMP[1].xxxx, TEMP[6].xxxx
 41: LRP TEMP[2].x, TEMP[1].xxxx, IN[0].yyyy, TEMP[2].xxxx
 42: LRP TEMP[0].x, TEMP[1].xxxx, TEMP[0].xxxx, IN[0].yyyy
 43: LRP TEMP[0].x, TEMP[1].xxxx, TEMP[0].xxxx, TEMP[2].xxxx
 44: LRP TEMP[1].xy, TEMP[1].xxxx, TEMP[4].xyyy, TEMP[3].xyyy
 45: ADD TEMP[2].x, TEMP[5].yyyy, -TEMP[5].xxxx
 46: RCP TEMP[3].x, TEMP[1].xxxx
 47: MUL TEMP[1].x, TEMP[1].yyyy, TEMP[3].xxxx
 48: MUL TEMP[1].x, TEMP[1].xxxx, TEMP[2].xxxx
 49: MOV TEMP[1].x, |TEMP[1].xxxx|
 50: MAD TEMP[3].x, TEMP[1].xxxx, IMM[0].yyyy, TEMP[0].xxxx
 51: MAD TEMP[4].x, TEMP[1].xxxx, IMM[0].xxxx, TEMP[0].xxxx
 52: MOV TEMP[3].y, TEMP[4].xxxx
 53: ADD TEMP[4].x, IMM[0].yyyy, -TEMP[0].xxxx
 54: RCP TEMP[5].x, TEMP[1].xxxx
 55: MUL TEMP[4].x, TEMP[4].xxxx, TEMP[5].xxxx
 56: MOV TEMP[3].z, TEMP[4].xxxx
 57: ADD TEMP[0].x, IMM[0].xxxx, -TEMP[0].xxxx
 58: RCP TEMP[1].x, TEMP[1].xxxx
 59: MUL TEMP[0].x, TEMP[0].xxxx, TEMP[1].xxxx
 60: MOV TEMP[3].w, TEMP[0].xxxx
 61: ADD TEMP[0], TEMP[3], IMM[0].yyyy
 62: MOV_SAT TEMP[0], TEMP[0]
 63: MUL TEMP[1].x, TEMP[0].zzzz, TEMP[0].yyyy
 64: ADD TEMP[1].x, TEMP[0].zzzz, -TEMP[1].xxxx
 65: ADD TEMP[1].x, TEMP[1].xxxx, IMM[0].zzzz
 66: ADD TEMP[1].x, TEMP[1].xxxx, -TEMP[0].xxxx
 67: MAD TEMP[0].x, TEMP[0].xxxx, TEMP[0].wwww, TEMP[1].xxxx
 68: MUL TEMP[0].x, IMM[0].yyyy, TEMP[0].xxxx
 69: MUL TEMP[0].x, TEMP[0].xxxx, TEMP[2].xxxx
 70: FSEQ TEMP[1].x, TEMP[2].xxxx, IMM[0].wwww
 71: UIF TEMP[1].xxxx
 72:   MOV TEMP[0].x, IMM[0].wwww
 73: ENDIF
 74: MOV TEMP[0].x, TEMP[0].xxxx
 75: MOV OUT[0], TEMP[0]
 76: END`, `VERT
DCL IN[0]
DCL IN[1]
DCL IN[2]
DCL IN[3]
DCL IN[4]
DCL OUT[0], POSITION
DCL OUT[1], GENERIC[9]
DCL OUT[2].xy, GENERIC[10]
DCL CONST[1][0..1]
DCL TEMP[0..5], LOCAL
IMM[0] UINT32 {0, 16, 0, 0}
IMM[1] FLT32 {    0.3750,    -0.5000,     1.0000,    -1.0000}
IMM[2] FLT32 {    0.1250,     0.0000,     0.0000,     0.0000}
  0: ADD TEMP[0].xy, IN[2].xyyy, CONST[1][1].xyyy
  1: ADD TEMP[1].xy, IN[3].xyyy, CONST[1][1].xyyy
  2: ADD TEMP[2].xy, IN[4].xyyy, CONST[1][1].xyyy
  3: ADD TEMP[3].x, IN[1].xxxx, CONST[1][1].yyyy
  4: MOV TEMP[4].x, IN[0].xxxx
  5: FSGE TEMP[5].x, IN[0].xxxx, IMM[1].xxxx
  6: UIF TEMP[5].xxxx
  7:   ADD TEMP[4].x, IN[0].xxxx, IMM[1].yyyy
  8:   ADD TEMP[3].x, TEMP[3].xxxx, IMM[1].zzzz
  9:   MOV TEMP[3].y, TEMP[3].xxxx
 10: ELSE
 11:   MIN TEMP[5].x, TEMP[0].yyyy, TEMP[1].yyyy
 12:   MIN TEMP[5].x, TEMP[5].xxxx, TEMP[2].yyyy
 13:   ADD TEMP[5].x, TEMP[5].xxxx, IMM[1].wwww
 14:   MOV TEMP[3].y, TEMP[5].xxxx
 15: ENDIF
 16: FSGE TEMP[4].x, TEMP[4].xxxx, IMM[2].xxxx
 17: UIF TEMP[4].xxxx
 18:   MAX TEMP[4].x, TEMP[0].xxxx, TEMP[1].xxxx
 19:   MAX TEMP[4].x, TEMP[4].xxxx, TEMP[2].xxxx
 20:   ADD TEMP[3].x, TEMP[4].xxxx, IMM[1].zzzz
 21: ELSE
 22:   MIN TEMP[4].x, TEMP[0].xxxx, TEMP[1].xxxx
 23:   MIN TEMP[4].x, TEMP[4].xxxx, TEMP[2].xxxx
 24:   ADD TEMP[3].x, TEMP[4].xxxx, IMM[1].wwww
 25: ENDIF
 26: ADD TEMP[0].xy, TEMP[0].xyyy, -TEMP[3].xyyy
 27: ADD TEMP[1].xy, TEMP[1].xyyy, -TEMP[3].xyyy
 28: ADD TEMP[2].xy, TEMP[2].xyyy, -TEMP[3].xyyy
 29: MAD TEMP[3].xy, TEMP[3].xyyy, CONST[1][0].xyyy, CONST[1][0].zwww
 30: MOV TEMP[4].zw, IMM[1].zzzz
 31: MOV TEMP[4].xy, TEMP[3].xyxx
 32: MOV OUT[2].xy, TEMP[2].xyxx
 33: MOV OUT[1].xy, TEMP[1].xyxx
 34: MOV OUT[1].zw, TEMP[0].xxxy
 35: MOV OUT[0], TEMP[4]
 36: END`},
}
