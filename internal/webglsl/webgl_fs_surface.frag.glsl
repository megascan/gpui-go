#version 300 es

precision highp float;
precision highp int;

struct GlobalParams {
    vec2 viewport_size;
    uint path_paint_base;
    uint path_curve_base;
    float depth_scale;
    uint quad_count;
    uint premultiplied_alpha;
    uint pad;
};
struct GammaParams {
    vec4 gamma_ratios;
    float grayscale_enhanced_contrast;
    float subpixel_enhanced_contrast;
    uint is_bgr;
    uint pad;
};
struct Bounds {
    vec2 origin;
    vec2 size;
};
struct Corners {
    float top_left;
    float top_right;
    float bottom_right;
    float bottom_left;
};
struct Edges {
    float top;
    float right;
    float bottom;
    float left;
};
struct Hsla {
    float h;
    float s;
    float l;
    float a;
};
struct LinearColorStop {
    Hsla color;
    float percentage;
};
struct Background {
    uint tag;
    uint color_space;
    Hsla solid;
    float gradient_angle_or_pattern_height;
    LinearColorStop colors[2];
    uint pad;
};
struct AtlasTextureId {
    uint index;
    uint kind;
};
struct AtlasBounds {
    ivec2 origin;
    ivec2 size;
};
struct AtlasTile {
    AtlasTextureId texture_id;
    uint tile_id;
    uint padding;
    AtlasBounds bounds;
};
struct TransformationMatrix {
    mat2x2 rotation_scale;
    vec2 translation;
};
struct ClippedVertex {
    vec2 position;
    vec2 unit_vertex;
};
struct PreparedBackground {
    vec4 color0_;
    vec4 color1_;
    uint tag;
};
struct Quad {
    uint order;
    uint border_style;
    Bounds bounds;
    Bounds content_mask;
    Background background;
    Hsla border_color;
    Corners corner_radii;
    Edges border_widths;
};
struct QuadVarying {
    vec4 position;
    vec4 border_color;
    uint quad_id;
    vec4 background_color0_;
    vec4 background_color1_;
    uint background_tag;
};
struct OpaqueQuadVarying {
    vec4 position;
    vec4 color;
};
struct Shadow {
    uint order;
    float blur_radius;
    Bounds bounds;
    Corners corner_radii;
    Bounds content_mask;
    Hsla color;
    Bounds shape_bounds;
    Corners shape_corner_radii;
    uint inset;
    uint pad;
};
struct ShadowVarying {
    vec4 position;
    vec4 color;
    uint shadow_id;
    vec4 blur_shape;
    vec4 blur_r1_;
    vec4 blur_exponent;
    vec4 blur_scale;
};
struct PreparedBlurredRoundedRect {
    vec4 shape;
    vec4 r1_;
    vec4 exponent;
    vec4 scale;
};
struct InstanceCursor {
    uint word_index;
    uint texel_index;
    uvec4 texel;
    uint width;
};
struct PathTile {
    uint paint;
    uint curve_start;
    uint curve_count;
    int backdrop;
    vec2 top_left;
    vec2 bottom_right;
    vec2 reference;
    vec2 curves_top_left;
    vec2 curves_bottom_right;
};
struct PathCurve {
    vec2 p0_;
    vec2 p1_;
    vec2 a;
    vec2 b;
    float sign_;
    uint pad;
};
struct PathPaint {
    Bounds bounds;
    Bounds content_mask;
    Background color;
    uint even_odd;
    uint pad;
};
struct PathTileVarying {
    vec4 position;
    uint tile_id;
    vec4 background_color0_;
    vec4 background_color1_;
    uint background_tag;
    vec4 curve_box;
};
struct Underline {
    uint order;
    uint pad;
    Bounds bounds;
    Bounds content_mask;
    Hsla color;
    float thickness;
    uint wavy;
};
struct UnderlineVarying {
    vec4 position;
    vec4 color;
    uint underline_id;
};
struct MonochromeSprite {
    uint order;
    uint pad;
    AtlasTile tile;
    Bounds bounds;
    Bounds content_mask;
    Hsla color;
    TransformationMatrix transformation;
};
struct MonoSpriteVarying {
    vec4 position;
    vec2 tile_position;
    vec4 color;
    vec4 clip_distances;
};
struct PolychromeSprite {
    uint order;
    uint pad;
    AtlasTile tile;
    uint grayscale;
    float opacity;
    Bounds bounds;
    Bounds content_mask;
    Corners corner_radii;
};
struct PolySpriteVarying {
    vec4 position;
    vec2 tile_position;
    uint sprite_id;
    float opacity;
};
struct SurfaceParams {
    Bounds bounds;
    Bounds content_mask;
};
struct SurfaceVarying {
    vec4 position;
    vec2 texture_position;
};
const float M_PI_F = 3.1415925;
const vec3 GRAYSCALE_FACTORS = vec3(0.2126, 0.7152, 0.0722);
const uint SIMPLE_PRIMITIVE_FLAG = 2147483648u;
const vec4 CULLED_POSITION = vec4(0.0, 0.0, 0.0, 1.0);
const mat4x4 ycbcr_to_RGB = mat4x4(vec4(1.0, 1.0, 1.0, 0.0), vec4(0.0, -0.3441, 1.772, 0.0), vec4(1.402, -0.7141, 0.0, 0.0), vec4(-0.701, 0.5291, -0.886, 1.0));

uniform highp sampler2D _group_1_binding_1_fs;

uniform highp sampler2D _group_1_binding_2_fs;

smooth in vec2 _vs2fs_location0;
layout(location = 0) out vec4 _fs2p_location0;

float color_brightness(vec3 color_2) {
    return dot(color_2, vec3(0.3, 0.59, 0.11));
}

float light_on_dark_contrast(float enhancedContrast, vec3 color_3) {
    float _e2 = color_brightness(color_3);
    float multiplier = clamp((4.0 * (0.75 - _e2)), 0.0, 1.0);
    return (enhancedContrast * multiplier);
}

float enhance_contrast(float alpha_1, float k) {
    return ((alpha_1 * (k + 1.0)) / ((alpha_1 * k) + 1.0));
}

vec3 enhance_contrast3_(vec3 alpha_2, float k_1) {
    return ((alpha_2 * (k_1 + 1.0)) / ((alpha_2 * k_1) + vec3(1.0)));
}

float apply_alpha_correction(float a, float b, vec4 g) {
    float brightness_adjustment = ((g.x * b) + g.y);
    float correction = ((brightness_adjustment * a) + ((g.z * b) + g.w));
    return (a + ((a * (1.0 - a)) * correction));
}

vec3 apply_alpha_correction3_(vec3 a_1, vec3 b_1, vec4 g_1) {
    vec3 brightness_adjustment_1 = ((g_1.x * b_1) + vec3(g_1.y));
    vec3 correction_1 = ((brightness_adjustment_1 * a_1) + ((g_1.z * b_1) + vec3(g_1.w)));
    return (a_1 + ((a_1 * (vec3(1.0) - a_1)) * correction_1));
}

float apply_contrast_and_gamma_correction(float sample_, vec3 color_4, float enhanced_contrast_factor, vec4 gamma_ratios) {
    float _e4 = light_on_dark_contrast(enhanced_contrast_factor, color_4);
    float _e5 = color_brightness(color_4);
    float _e6 = enhance_contrast(sample_, _e4);
    float _e7 = apply_alpha_correction(_e6, _e5, gamma_ratios);
    return _e7;
}

vec3 apply_contrast_and_gamma_correction3_(vec3 sample_1, vec3 color_5, float enhanced_contrast_factor_1, vec4 gamma_ratios_1) {
    float _e4 = light_on_dark_contrast(enhanced_contrast_factor_1, color_5);
    vec3 _e5 = enhance_contrast3_(sample_1, _e4);
    vec3 _e6 = apply_alpha_correction3_(_e5, color_5, gamma_ratios_1);
    return _e6;
}

vec4 distance_from_clip_rect_impl(vec2 position_1, Bounds clip_bounds) {
    vec2 tl = (position_1 - clip_bounds.origin);
    vec2 br = ((clip_bounds.origin + clip_bounds.size) - position_1);
    return vec4(tl.x, br.x, tl.y, br.y);
}

vec4 distance_from_clip_rect_transformed(vec2 unit_vertex_3, Bounds bounds_2, Bounds clip_bounds_1, TransformationMatrix transform_1) {
    vec2 position_4 = ((unit_vertex_3 * vec2(bounds_2.size)) + bounds_2.origin);
    vec2 transformed = ((transpose(transform_1.rotation_scale) * position_4) + transform_1.translation);
    vec4 _e14 = distance_from_clip_rect_impl(transformed, clip_bounds_1);
    return _e14;
}

ClippedVertex clip_to_mask(vec2 unit_vertex_4, Bounds bounds_3, Bounds mask) {
    ClippedVertex result = ClippedVertex(vec2(0.0), vec2(0.0));
    vec2 origin = max(bounds_3.origin, mask.origin);
    vec2 corner = min((bounds_3.origin + bounds_3.size), (mask.origin + mask.size));
    vec2 size_1 = max((corner - origin), vec2(0.0));
    result.position = (origin + (unit_vertex_4 * size_1));
    vec2 _e23 = result.position;
    result.unit_vertex = ((_e23 - bounds_3.origin) / bounds_3.size);
    ClippedVertex _e28 = result;
    return _e28;
}

vec3 srgb_to_linear(vec3 srgb) {
    bvec3 cutoff = lessThan(srgb, vec3(0.04045));
    vec3 higher = pow(((srgb + vec3(0.055)) / vec3(1.055)), vec3(2.4));
    vec3 lower = (srgb / vec3(12.92));
    return mix(higher, lower, cutoff);
}

float srgb_to_linear_component(float a_2) {
    bool cutoff_1 = (a_2 < 0.04045);
    float higher_1 = pow(((a_2 + 0.055) / 1.055), 2.4);
    float lower_1 = (a_2 / 12.92);
    return (cutoff_1 ? lower_1 : higher_1);
}

vec3 linear_to_srgb(vec3 linear) {
    bvec3 cutoff_2 = lessThan(linear, vec3(0.0031308));
    vec3 higher_2 = ((vec3(1.055) * pow(linear, vec3(0.41666666))) - vec3(0.055));
    vec3 lower_2 = (linear * vec3(12.92));
    return mix(higher_2, lower_2, cutoff_2);
}

vec4 linear_to_srgba(vec4 color_6) {
    vec3 _e2 = linear_to_srgb(color_6.xyz);
    return vec4(_e2, color_6.w);
}

vec4 srgba_to_linear(vec4 color_7) {
    vec3 _e2 = srgb_to_linear(color_7.xyz);
    return vec4(_e2, color_7.w);
}

vec4 hsla_to_rgba(Hsla hsla) {
    vec3 color_8 = vec3(0.0);
    bool local_9 = false;
    bool local_10 = false;
    bool local_11 = false;
    bool local_12 = false;
    bool local_13 = false;
    float h = (hsla.h * 6.0);
    float s = hsla.s;
    float l = hsla.l;
    float a_5 = hsla.a;
    float c_1 = ((1.0 - abs(((2.0 * l) - 1.0))) * s);
    float x = (c_1 * (1.0 - abs(((h - 2.0 * trunc(h / 2.0)) - 1.0))));
    float m = (l - (c_1 / 2.0));
    color_8 = vec3(m);
    if ((h >= 0.0)) {
        local_9 = (h < 1.0);
    } else {
        local_9 = false;
    }
    bool _e35 = local_9;
    if (_e35) {
        float _e37 = color_8.x;
        color_8.x = (_e37 + c_1);
        float _e40 = color_8.y;
        color_8.y = (_e40 + x);
    } else {
        if ((h >= 1.0)) {
            local_10 = (h < 2.0);
        } else {
            local_10 = false;
        }
        bool _e49 = local_10;
        if (_e49) {
            float _e51 = color_8.x;
            color_8.x = (_e51 + x);
            float _e54 = color_8.y;
            color_8.y = (_e54 + c_1);
        } else {
            if ((h >= 2.0)) {
                local_11 = (h < 3.0);
            } else {
                local_11 = false;
            }
            bool _e63 = local_11;
            if (_e63) {
                float _e65 = color_8.y;
                color_8.y = (_e65 + c_1);
                float _e68 = color_8.z;
                color_8.z = (_e68 + x);
            } else {
                if ((h >= 3.0)) {
                    local_12 = (h < 4.0);
                } else {
                    local_12 = false;
                }
                bool _e77 = local_12;
                if (_e77) {
                    float _e79 = color_8.y;
                    color_8.y = (_e79 + x);
                    float _e82 = color_8.z;
                    color_8.z = (_e82 + c_1);
                } else {
                    if ((h >= 4.0)) {
                        local_13 = (h < 5.0);
                    } else {
                        local_13 = false;
                    }
                    bool _e91 = local_13;
                    if (_e91) {
                        float _e93 = color_8.x;
                        color_8.x = (_e93 + x);
                        float _e96 = color_8.z;
                        color_8.z = (_e96 + c_1);
                    } else {
                        float _e99 = color_8.x;
                        color_8.x = (_e99 + c_1);
                        float _e102 = color_8.z;
                        color_8.z = (_e102 + x);
                    }
                }
            }
        }
    }
    vec3 _e104 = color_8;
    return vec4(_e104, a_5);
}

vec4 linear_srgb_to_oklab(vec4 color_9) {
    float l_1 = (((0.41222146 * color_9.x) + (0.53633255 * color_9.y)) + (0.051445995 * color_9.z));
    float m_1 = (((0.2119035 * color_9.x) + (0.6806995 * color_9.y)) + (0.10739696 * color_9.z));
    float s_1 = (((0.08830246 * color_9.x) + (0.28171885 * color_9.y)) + (0.6299787 * color_9.z));
    float l_2 = pow(l_1, 0.33333334);
    float m_2 = pow(m_1, 0.33333334);
    float s_2 = pow(s_1, 0.33333334);
    return vec4((((0.21045426 * l_2) + (0.7936178 * m_2)) - (0.004072047 * s_2)), (((1.9779985 * l_2) - (2.4285922 * m_2)) + (0.4505937 * s_2)), (((0.025904037 * l_2) + (0.78277177 * m_2)) - (0.80867577 * s_2)), color_9.w);
}

vec4 oklab_to_linear_srgb(vec4 color_10) {
    float l_3 = ((color_10.x + (0.39633778 * color_10.y)) + (0.21580376 * color_10.z));
    float m_3 = ((color_10.x - (0.105561346 * color_10.y)) - (0.06385417 * color_10.z));
    float s_3 = ((color_10.x - (0.08948418 * color_10.y)) - (1.2914855 * color_10.z));
    float l_4 = ((l_3 * l_3) * l_3);
    float m_4 = ((m_3 * m_3) * m_3);
    float s_4 = ((s_3 * s_3) * s_3);
    return vec4((((4.0767417 * l_4) - (3.3077116 * m_4)) + (0.23096994 * s_4)), (((-1.268438 * l_4) + (2.6097574 * m_4)) - (0.34131938 * s_4)), (((-0.0041960864 * l_4) - (0.7034186 * m_4)) + (1.7076147 * s_4)), color_10.w);
}

vec4 over(vec4 below, vec4 above) {
    float alpha_3 = (above.w + (below.w * (1.0 - above.w)));
    vec3 premultiplied = ((above.xyz * above.w) + ((below.xyz * below.w) * (1.0 - above.w)));
    return vec4((premultiplied / vec3(((alpha_3 == 0.0) ? 1.0 : alpha_3))), alpha_3);
}

float pick_corner_radius(vec2 center_to_point, Corners radii) {
    if ((center_to_point.x < 0.0)) {
        if ((center_to_point.y < 0.0)) {
            return radii.top_left;
        } else {
            return radii.bottom_left;
        }
    } else {
        if ((center_to_point.y < 0.0)) {
            return radii.top_right;
        } else {
            return radii.bottom_right;
        }
    }
}

float quad_sdf_impl(vec2 corner_center_to_point, float corner_radius) {
    if ((corner_radius == 0.0)) {
        return max(corner_center_to_point.x, corner_center_to_point.y);
    } else {
        float signed_distance_to_inset_quad = (length(max(vec2(0.0), corner_center_to_point)) + min(0.0, max(corner_center_to_point.x, corner_center_to_point.y)));
        return (signed_distance_to_inset_quad - corner_radius);
    }
}

float quad_sdf(vec2 point, Bounds bounds_4, Corners corner_radii) {
    vec2 half_size = (bounds_4.size / vec2(2.0));
    vec2 center_2 = (bounds_4.origin + half_size);
    vec2 center_to_point_3 = (point - center_2);
    float _e10 = pick_corner_radius(center_to_point_3, corner_radii);
    vec2 corner_to_point = (abs(center_to_point_3) - half_size);
    vec2 corner_center_to_point_1 = (corner_to_point + vec2(_e10));
    float _e15 = quad_sdf_impl(corner_center_to_point_1, _e10);
    return _e15;
}

PreparedBackground prepare_background(Background background) {
    PreparedBackground result_1 = PreparedBackground(vec4(0.0), vec4(0.0), 0u);
    result_1.tag = background.tag;
    result_1.color1_ = vec4(0.0);
    if ((background.tag == 1u)) {
        vec4 _e14 = hsla_to_rgba(background.colors[0].color);
        result_1.color0_ = _e14;
        vec4 _e19 = hsla_to_rgba(background.colors[1].color);
        result_1.color1_ = _e19;
        if ((background.color_space == 0u)) {
            vec4 _e25 = result_1.color0_;
            vec4 _e26 = linear_to_srgba(_e25);
            result_1.color0_ = _e26;
            vec4 _e29 = result_1.color1_;
            vec4 _e30 = linear_to_srgba(_e29);
            result_1.color1_ = _e30;
        } else {
            if ((background.color_space == 1u)) {
                vec4 _e36 = result_1.color0_;
                vec4 _e37 = linear_srgb_to_oklab(_e36);
                result_1.color0_ = _e37;
                vec4 _e40 = result_1.color1_;
                vec4 _e41 = linear_srgb_to_oklab(_e40);
                result_1.color1_ = _e41;
            }
        }
    } else {
        vec4 _e44 = hsla_to_rgba(background.solid);
        result_1.color0_ = _e44;
    }
    PreparedBackground _e45 = result_1;
    return _e45;
}

float dither_offset(vec2 position_2) {
    uvec2 cell = (uvec2(position_2) & uvec2(3u));
    uint z = (cell.x ^ cell.y);
    uint threshold = (((((z & 1u) << 3u) | ((cell.y & 1u) << 2u)) | (z & 2u)) | ((cell.y & 2u) >> 1u));
    return (((float(threshold) + 0.5) * 0.125) - 1.0);
}

vec4 background_color(PreparedBackground prepared, Background background_1, Bounds bounds_5, vec2 position_3) {
    vec4 color_12 = vec4(0.0);
    vec2 direction = vec2(0.0);
    vec3 gradient = vec3(0.0);
    bool local_14 = false;
    color_12 = prepared.color0_;
    if ((prepared.tag == 0u)) {
        vec4 _e9 = color_12;
        return _e9;
    }
    switch(prepared.tag) {
        case 1u: {
            float radians_ = ((((background_1.gradient_angle_or_pattern_height - 360.0 * trunc(background_1.gradient_angle_or_pattern_height / 360.0)) - 90.0) * M_PI_F) / 180.0);
            direction = vec2(cos(radians_), sin(radians_));
            if ((bounds_5.size.x > bounds_5.size.y)) {
                float _e30 = direction.y;
                direction.y = (_e30 * (bounds_5.size.y / bounds_5.size.x));
            } else {
                float _e38 = direction.x;
                direction.x = (_e38 * (bounds_5.size.x / bounds_5.size.y));
            }
            float _e50 = direction.x;
            float _e53 = direction.y;
            float extent = ((abs(_e50) > abs(_e53)) ? bounds_5.size.x : bounds_5.size.y);
            vec2 _e57 = direction;
            float direction_length = length(_e57);
            float span = (background_1.colors[1].percentage - background_1.colors[0].percentage);
            float offset = (0.5 - background_1.colors[0].percentage);
            if ((direction_length != 0.0)) {
                local_14 = (extent != 0.0);
            } else {
                local_14 = false;
            }
            bool _e81 = local_14;
            if (_e81) {
                if ((span == 0.0)) {
                    vec2 _e85 = direction;
                    gradient = vec3(((_e85 / vec2((direction_length * extent))) * 1e30), (offset * 1e30));
                } else {
                    vec2 _e92 = direction;
                    gradient = vec3((_e92 / vec2(((direction_length * extent) * span))), (offset / span));
                }
            }
            vec2 center_3 = (bounds_5.origin + (bounds_5.size * 0.5));
            vec3 _e105 = gradient;
            float _e109 = gradient.z;
            float t_2 = clamp((dot((position_3 - center_3), _e105.xy) + _e109), 0.0, 1.0);
            color_12 = mix(prepared.color0_, prepared.color1_, t_2);
            if ((background_1.color_space == 1u)) {
                vec4 _e118 = color_12;
                vec4 _e119 = oklab_to_linear_srgb(_e118);
                color_12 = _e119;
            } else {
                vec4 _e120 = color_12;
                vec4 _e121 = srgba_to_linear(_e120);
                color_12 = _e121;
            }
            float _e122 = dither_offset(position_3);
            vec4 _e123 = color_12;
            float _e132 = color_12.w;
            color_12 = vec4((_e123.xyz + vec3(((_e122 * 1.0) / 255.0))), (_e132 + ((_e122 * 1.5) / 255.0)));
            break;
        }
        case 2u: {
            float pattern_height_encoded = background_1.gradient_angle_or_pattern_height;
            float pattern_width = ((pattern_height_encoded / 65535.0) / 255.0);
            float pattern_interval = ((pattern_height_encoded - 65535.0 * trunc(pattern_height_encoded / 65535.0)) / 255.0);
            float pattern_height = (pattern_width + pattern_interval);
            vec2 stripe_direction = vec2(cos(0.7853981), sin(0.7853981));
            float pattern_period = (pattern_height * sin(0.7853981));
            float threshold_1 = ((pattern_period * (pattern_width / pattern_height)) / 2.0);
            float pattern = (dot((position_3 - bounds_5.origin), stripe_direction) - pattern_period * trunc(dot((position_3 - bounds_5.origin), stripe_direction) / pattern_period));
            float distance_ = (min(pattern, (pattern_period - pattern)) - threshold_1);
            float _e167 = color_12.w;
            color_12.w = (_e167 * clamp((0.5 - distance_), 0.0, 1.0));
            break;
        }
        case 3u: {
            float cell_size = background_1.gradient_angle_or_pattern_height;
            vec2 relative_position = (position_3 - bounds_5.origin);
            float x_index = floor((relative_position.x / cell_size));
            float y_index = floor((relative_position.y / cell_size));
            float _e182 = color_12.w;
            color_12.w = (_e182 * clamp(((x_index + y_index) - 2.0 * trunc((x_index + y_index) / 2.0)), 0.0, 1.0));
            break;
        }
        default: {
            break;
        }
    }
    vec4 _e188 = color_12;
    return _e188;
}

bool has_rounded_corners(Corners corner_radii_1) {
    bool local_15 = false;
    bool local_16 = false;
    bool local_17 = false;
    if (!((corner_radii_1.top_left != 0.0))) {
        local_15 = (corner_radii_1.top_right != 0.0);
    } else {
        local_15 = true;
    }
    bool _e11 = local_15;
    if (!(_e11)) {
        local_16 = (corner_radii_1.bottom_right != 0.0);
    } else {
        local_16 = true;
    }
    bool _e19 = local_16;
    if (!(_e19)) {
        local_17 = (corner_radii_1.bottom_left != 0.0);
    } else {
        local_17 = true;
    }
    bool _e27 = local_17;
    return _e27;
}

bool has_border(Edges border_widths) {
    bool local_18 = false;
    bool local_19 = false;
    bool local_20 = false;
    if (!((border_widths.top != 0.0))) {
        local_18 = (border_widths.right != 0.0);
    } else {
        local_18 = true;
    }
    bool _e11 = local_18;
    if (!(_e11)) {
        local_19 = (border_widths.bottom != 0.0);
    } else {
        local_19 = true;
    }
    bool _e19 = local_19;
    if (!(_e19)) {
        local_20 = (border_widths.left != 0.0);
    } else {
        local_20 = true;
    }
    bool _e27 = local_20;
    return _e27;
}

bool is_shaped(Quad quad) {
    bool local_21 = false;
    bool _e2 = has_rounded_corners(quad.corner_radii);
    if (!(_e2)) {
        bool _e7 = has_border(quad.border_widths);
        local_21 = _e7;
    } else {
        local_21 = true;
    }
    bool _e9 = local_21;
    return _e9;
}

bool is_opaque_rectangle(Quad quad_1) {
    bool local_22 = false;
    bool local_23 = false;
    bool local_24 = false;
    if ((quad_1.background.tag == 0u)) {
        local_22 = (quad_1.background.solid.a >= 1.0);
    } else {
        local_22 = false;
    }
    bool _e13 = local_22;
    if (_e13) {
        bool _e17 = has_rounded_corners(quad_1.corner_radii);
        local_23 = !(_e17);
    } else {
        local_23 = false;
    }
    bool _e20 = local_23;
    if (_e20) {
        bool _e24 = has_border(quad_1.border_widths);
        local_24 = !(_e24);
    } else {
        local_24 = false;
    }
    bool _e27 = local_24;
    return _e27;
}

float quarter_ellipse_sdf(vec2 point_1, vec2 radii_1) {
    vec2 circle_vec = (point_1 / radii_1);
    float unit_circle_sdf = (length(circle_vec) - 1.0);
    return ((unit_circle_sdf * (radii_1.x + radii_1.y)) * -0.5);
}

float corner_dash_velocity(float dv1_, float dv2_) {
    if ((dv1_ == 0.0)) {
        return dv2_;
    } else {
        if ((dv2_ == 0.0)) {
            return dv1_;
        } else {
            return min(dv1_, dv2_);
        }
    }
}

float fmod(float a_3, float b_2) {
    return (a_3 - (b_2 * trunc((a_3 / b_2))));
}

float dash_alpha(float t_1, float period, float length_, float dash_velocity_1, float antialias_threshold) {
    float half_period = (period / 2.0);
    float half_length = (length_ / 2.0);
    float _e11 = fmod(((t_1 + half_period) - half_length), period);
    float centered = (_e11 - half_period);
    float signed_distance = (abs(centered) - half_length);
    return clamp((antialias_threshold - (signed_distance / dash_velocity_1)), 0.0, 1.0);
}

float erf7_(float input_8) {
    float x_1 = (clamp(input_8, -10.0, 10.0) * 1.1283792);
    float xx = (x_1 * x_1);
    float y = (x_1 + ((0.24295 + ((0.03395 + (0.0104 * xx)) * xx)) * (x_1 * xx)));
    return (y / sqrt((1.0 + (y * y))));
}

PreparedBlurredRoundedRect prepare_blurred_rounded_rect(vec2 size, Corners corner_radii_2, float sigma) {
    PreparedBlurredRoundedRect output_ = PreparedBlurredRoundedRect(vec4(0.0), vec4(0.0), vec4(0.0), vec4(0.0));
    float s_5 = max((sigma * 1.4142135), 1e-6);
    float s_inv = (1.0 / s_5);
    float min_edge = min(size.x, size.y);
    float r_max = (0.5 * min_edge);
    vec4 radii_2 = vec4(corner_radii_2.top_left, corner_radii_2.top_right, corner_radii_2.bottom_right, corner_radii_2.bottom_left);
    vec4 r0_ = min(sqrt(((radii_2 * radii_2) + vec4(((1.3225 * s_5) * s_5)))), vec4(r_max));
    vec4 r1_ = min(sqrt(((radii_2 * radii_2) + vec4(((4.0 * s_5) * s_5)))), vec4(r_max));
    vec2 half_extent = ((0.5 * s_inv) * size);
    float delta = ((1.25 * s_5) * (exp((-(half_extent.x) * half_extent.x)) - exp((-(half_extent.y) * half_extent.y))));
    vec2 adjusted_size = vec2((size.x + min(delta, 0.0)), (size.y - max(delta, 0.0)));
    vec4 peak = ((s_inv * 0.5) * (vec4(max(adjusted_size.x, adjusted_size.y)) - (0.5 * radii_2)));
    output_.shape = vec4((0.5 * adjusted_size), s_inv, min_edge);
    output_.r1_ = r1_;
    output_.exponent = ((2.0 * r1_) / r0_);
    float _e85 = erf7_(peak.x);
    float _e87 = erf7_(peak.y);
    float _e89 = erf7_(peak.z);
    float _e91 = erf7_(peak.w);
    output_.scale = (0.5 * vec4(_e85, _e87, _e89, _e91));
    PreparedBlurredRoundedRect _e95 = output_;
    return _e95;
}

float pick_corner(vec2 center_to_point_1, vec4 values) {
    float _e7 = pick_corner_radius(center_to_point_1, Corners(values.x, values.y, values.z, values.w));
    return _e7;
}

float blurred_rounded_rect(vec2 center_to_point_2, PreparedBlurredRoundedRect prepared_1) {
    float d_pos = 0.0;
    bool local_25 = false;
    float _e3 = pick_corner(center_to_point_2, prepared_1.r1_);
    float _e5 = pick_corner(center_to_point_2, prepared_1.exponent);
    float _e7 = pick_corner(center_to_point_2, prepared_1.scale);
    float s_inv_1 = prepared_1.shape.z;
    float min_edge_1 = prepared_1.shape.w;
    vec2 p0_ = (abs(center_to_point_2) - (prepared_1.shape.xy - vec2(_e3)));
    vec2 p1_ = max(p0_, vec2(0.0));
    if ((p1_.x > 0.0)) {
        local_25 = (p1_.y > 0.0);
    } else {
        local_25 = false;
    }
    bool _e31 = local_25;
    if (_e31) {
        d_pos = pow((pow(p1_.x, _e5) + pow(p1_.y, _e5)), (1.0 / _e5));
    } else {
        d_pos = (p1_.x + p1_.y);
    }
    float d_neg = min(max(p0_.x, p0_.y), 0.0);
    float _e48 = d_pos;
    float d = ((_e48 + d_neg) - _e3);
    float _e53 = erf7_((s_inv_1 * (min_edge_1 + d)));
    float _e55 = erf7_((s_inv_1 * d));
    return (_e7 * (_e53 - _e55));
}

float monotone_quadratic_root(float a_4, float b_3, float c, float direction_1) {
    float discriminant = max(((b_3 * b_3) - ((4.0 * a_4) * c)), 0.0);
    float denominator = (b_3 + (direction_1 * sqrt(discriminant)));
    if ((denominator == 0.0)) {
        return 0.0;
    }
    return clamp(((-2.0 * c) / denominator), 0.0, 1.0);
}

void add_curve_winding(PathCurve curve, vec2 center, inout vec2 winding_1, inout vec2 weight_1) {
    bool local_26 = false;
    bool local_27 = false;
    float min_x = min(curve.p0_.x, curve.p1_.x);
    float max_x = max(curve.p0_.x, curve.p1_.x);
    if ((curve.p0_.y <= center.y)) {
        local_26 = (center.y < curve.p1_.y);
    } else {
        local_26 = false;
    }
    bool _e25 = local_26;
    if (_e25) {
        if ((max_x <= (center.x - 0.5))) {
            float _e31 = winding_1.x;
            winding_1.x = (_e31 + curve.sign_);
        } else {
            float _e43 = monotone_quadratic_root(curve.a.y, curve.b.y, (curve.p0_.y - center.y), 1.0);
            float x_2 = ((((curve.a.x * _e43) + curve.b.x) * _e43) + curve.p0_.x);
            float distance_1 = (center.x - x_2);
            float _e57 = winding_1.x;
            winding_1.x = (_e57 + (curve.sign_ * clamp((distance_1 + 0.5), 0.0, 1.0)));
            float _e66 = weight_1.x;
            weight_1.x = max(_e66, clamp((1.0 - (2.0 * abs(distance_1))), 0.0, 1.0));
        }
    }
    if ((min_x <= center.x)) {
        local_27 = (center.x < max_x);
    } else {
        local_27 = false;
    }
    bool _e81 = local_27;
    if (_e81) {
        float direction_2 = ((curve.p1_.x > curve.p0_.x) ? 1.0 : -1.0);
        if ((curve.p1_.y <= (center.y - 0.5))) {
            float _e97 = winding_1.y;
            winding_1.y = (_e97 - (curve.sign_ * direction_2));
            return;
        } else {
            if ((curve.p0_.y < (center.y + 0.5))) {
                float _e115 = monotone_quadratic_root(curve.a.x, curve.b.x, (curve.p0_.x - center.x), direction_2);
                float y_1 = ((((curve.a.y * _e115) + curve.b.y) * _e115) + curve.p0_.y);
                float distance_2 = (center.y - y_1);
                float _e129 = winding_1.y;
                winding_1.y = (_e129 - ((curve.sign_ * direction_2) * clamp((distance_2 + 0.5), 0.0, 1.0)));
                float _e139 = weight_1.y;
                weight_1.y = max(_e139, clamp((1.0 - (2.0 * abs(distance_2))), 0.0, 1.0));
                return;
            } else {
                return;
            }
        }
    } else {
        return;
    }
}

void correct_for_endpoint(vec2 endpoint, float step_, vec2 reference, vec2 center_1, inout vec2 winding_2) {
    bool local_28 = false;
    if ((endpoint.x <= reference.x)) {
        float _e9 = winding_2.x;
        winding_2.x = (_e9 - (step_ * (float((endpoint.y <= center_1.y)) - float((endpoint.y <= reference.y)))));
    }
    if ((endpoint.y <= reference.y)) {
        local_28 = (endpoint.x <= center_1.x);
    } else {
        local_28 = false;
    }
    bool _e30 = local_28;
    if (_e30) {
        float _e32 = winding_2.y;
        winding_2.y = (_e32 + step_);
        return;
    } else {
        return;
    }
}

void main() {
    SurfaceVarying input_7 = SurfaceVarying(gl_FragCoord, _vs2fs_location0);
    vec4 _e5 = textureLod(_group_1_binding_1_fs, vec2(input_7.texture_position), 0.0);
    vec4 _e11 = textureLod(_group_1_binding_2_fs, vec2(input_7.texture_position), 0.0);
    vec4 y_cb_cr = vec4(_e5.x, _e11.xy, 1.0);
    _fs2p_location0 = (ycbcr_to_RGB * y_cb_cr);
    return;
}

