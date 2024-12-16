# Add-on for Tetra3D > Blender exporting

import bpy, os, bmesh, math, mathutils, aud, bpy_extras
from bpy.app.handlers import persistent

currentlyPlayingAudioName = None
currentlyPlayingAudioHandle = None
audioPaused = False

bl_info = {
    "name" : "Tetra3D Addon",                        # The name in the addon search menu
    "author" : "SolarLune Games",
    "description" : "An addon for exporting GLTF content from Blender for use with Tetra3D.",
    "blender" : (3, 0, 1),                             # Lowest version to use
    "location" : "View3D",
    "category" : "Gamedev",
    "version" : (0, 16, 0),
    "support" : "COMMUNITY",
    "doc_url" : "https://github.com/SolarLune/Tetra3d/wiki/Blender-Addon",
}

objectTypes = [
    ("MESH", "Mesh", "A standard, visible mesh object.", 0, 0),
    ("GRID", "Grid", "A grid object; not visualized or 'physically present'. The vertices in Blender become grid points in Tetra3D; the edges become their connections.", 0, 1),
]

sectorTypes = [
    ("OBJECT", "Object", "An Object that lies in a sector will only render if the sector it is in also renders.", 0, 0),
    ("SECTOR", "Sector", "A Sector indicates a space that contains objects and maintains neighborly relationships with other Sectors.", 0, 1),
    ("STANDALONE", "Standalone", "A Standalone object will render regardless of if the sector it lies in renders (or if it lies in any sector at all).", 0, 2),
]

def listSectorTypes(self, context):

    if context.object.type == "MESH" or (context.object.instance_type == "COLLECTION" and context.object.instance_collection is not None):
        return sectorTypes
    return sectorTypes[::2]

boundsTypes = [
    ("NONE", "No Bounds", "No collision will be created for this object", 0, 0),
    ("AABB", "AABB", "An AABB (axis-aligned bounding box). If the size isn't customized, it will be big enough to fully contain the mesh of the current object. Currently buggy when resolving intersections between AABB or other Triangle Nodes", 0, 1),
    ("CAPSULE", "Capsule", "A capsule, which can rotate. If the radius and height are not set, it will have a radius and height to fully contain the current object", 0, 2),
    ("SPHERE", "Sphere", "A sphere. If the radius is not custom set, it will have a large enough radius to fully contain the provided object", 0, 3),
    ("TRIANGLES", "Triangle Mesh", "A triangle mesh bounds type. Only works on mesh-type objects (i.e. an Empty won't generate a BoundingTriangles). Accurate, but slow. Currently buggy when resolving intersections between AABB or other Triangle Nodes", 0, 4),
]

gltfExportTypes = [
    ("GLB", ".glb", "Exports a single file, with all data packed in binary form. Textures can be packed into the file. Most efficient and portable, but more difficult to edit later", 0, 0),
    ("GLTF_SEPARATE", ".gltf + .bin + external textures", "Exports multiple files, with separate JSON, binary and texture data. Easiest to edit later", 0, 1),
]

sectorDetectionType = [
    ("VERTICES", "Vertices", "Sector neighborhood is determined by sectors sharing vertex positions. Accurate, but slower", 0, 0),
    ("AABB", "AABB", "Sector neighborhood is determined by AABB intersection. Fast, but inaccurate", 0, 1),
]

materialBlendModes = [
    ("DEFAULT", "Default", "Blends the destination by the material's color modulated by the material's alpha value. The default alpha-blending composite mode. Also known as BlendSourceOver", 0, 0),
    ("ADDITIVE", "Additive", "Adds the material's color to the destination. Also known as BlendLighter", 0, 1),
    ("MULTIPLY", "Multiply", "Multiplies the material's color by the destination. Known as Multiply compositing using a custom Blend object", 0, 2),
    ("CLEAR", "Clear", "Anywhere the material draws is cleared instead; useful to 'punch through' a scene to show the blank alpha zero. Also known as BlendClear", 0, 3),
]

materialTransparencyModes = [
    ("AUTO", "Auto", "The material is opaque until its material color has an alpha below 1; in this case, it switches to Transparent mode", 0, 0),
    ("OPAQUE", "Opaque", "The material is wholly opaque", 0, 1),
    ("ALPHA CLIP", "Alpha Clip", "Transparency is determined by the texture's alpha channel and is either wholly transparent or wholly opaque. Renders after all opaque objects and is sorted from back-to-front", 0, 2),
    ("TRANSPARENT", "Transparent", "Partial transparency. Renders after all opaque objects and is sorted from back-to-front", 0, 3),
]

materialBillboardModes = [
    ("NONE", "None", "No billboarding - the (unskinned) object with this material does not rotate to face the camera.", 0, 0),
    ("FIXEDVERTICAL", "Fixed Vertical", "Fixed Vertical billboarding - the (unskinned) object with this material faces the camera, with up always pointing towards the camera's local up vector (+Y). Good for top-down games.", 0, 1),
    ("HORIZONTAL", "Horizontal", "Horizontal billboarding - the (unskinned) object with this material rotates around to the face the camera only on the X and Z axes (not the Y axis).", 0, 2),
    ("FULL", "Full", "Full billboarding - the (unskinned) object rotates fully to face the camera.", 0, 3),
]

materialLightingModes = [
    ("DEFAULT", "Default", "Default lighting; light is dependent on normal of lit faces.", 0, 0),
    ("NORMAL", "Point Towards Lights", "Lighting acts as though faces always face light sources. Particularly useful for billboarded 2D sprites.", 0, 1),
    ("DOUBLE", "Double-Sided", "Double-sided lighting; lighting is dependent on normal of lit faces, but on both sides of a face.", 0, 2),
]

worldFogCompositeModes = [
    ("OFF", "Off", "No fog. Object colors aren't changed with distance from the camera.", 0, 0),
    ("ADDITIVE", "Additive", "Additive fog - this fog mode brightens objects in the distance, with full effect being adding the color given to the object's color at maximum distance (according to the camera's far range).", 0, 1),
    ("SUBTRACT", "Subtractive", "Subtractive fog - this fog mode darkens objects in the distance, with full effect being subtracting the object's color by the fog color at maximum distance (according to the camera's far range).", 0, 2),
    ("OVERWRITE", "Overwrite", "Overwrite fog - this fog mode overwrites the object's color with the fog color, with maximum distance being the camera's far distance.", 0, 3),
    ("TRANSPARENT", "Transparent", "Transparent fog - this fog mode fades the object out over distance, such that at maximum distance / fog range, the object is wholly transparent.", 0, 4),
]

worldFogCurveTypes = [
    ("LINEAR", "Smooth", "Smooth fog (Ease: Linear); this goes from 0% in the near range to 100% in the far range evenly.", "LINCURVE", 0),
    ("OUTCIRC", "Dense", "Dense fog (Ease: Out Circ); fog will increase aggressively in the near range, ramping up to 100% at the far range.", "SPHERECURVE", 1),
    ("INCIRC", "Light", "Light fog (Ease: In Circ); fog will increase aggressively towards the far range, ramping up to 100% at the far range.", "SHARPCURVE", 2),
]

gamePropTypes = [
    ("bool", "Bool", "Boolean data type", 0, 0),
    ("int", "Int", "Int data type", 0, 1),
    ("float", "Float", "Float data type", 0, 2),
    ("string", "String", "String data type", 0, 3),
    ("reference", "Object", "Object reference data type; converted to a string composed as follows on export - [SCENE NAME]:[OBJECT NAME]", 0, 4),
    ("color", "Color", "Color data type", 0, 5),
    ("vector3d", "3D Vector", "3D vector data type", 0, 6),
    ("file", "Filepath", "Filepath as a string", 0, 7),
    ("directory", "Directory Path", "Directory Path as a string", 0, 8),
]

batchModes = [ 
    ("OFF", "Off", "No automatic batching.", 0, 0), 
    ("DYNAMIC", "Dynamic Batching", "Dynamic batching based off of one material (the first one).", 0, 1), 
    ("STATIC", "Static Merging", "Static merging; merged objects cannot move or deviate in any way. After automatic static merging, the merged models will be automatically set to invisible.", 0, 2),
]

def filepathSet(self, value):
    global currentlyPlayingAudioHandle, currentlyPlayingAudioName, audioPaused
    if "valueFilepath" in self and self["valueFilepath"] == currentlyPlayingAudioName and value != self["valueFilepath"]:
        currentlyPlayingAudioHandle.stop()
        currentlyPlayingAudioHandle = None
        currentlyPlayingAudioName = ""
        audioPaused = False
    self["valueFilepath"] = value

def filepathGet(self):
    if "valueFilepath" in self:
        return self["valueFilepath"]
    return ""


class t3dGamePropertyItem__(bpy.types.PropertyGroup):

    name: bpy.props.StringProperty(name="Name", default="New Property")
    valueType: bpy.props.EnumProperty(items=gamePropTypes, name="Type")

    valueBool: bpy.props.BoolProperty(name = "", description="The boolean value of the property")
    valueInt: bpy.props.IntProperty(name = "", description="The integer value of the property")
    valueFloat: bpy.props.FloatProperty(name = "", description="The float value of the property")
    valueString: bpy.props.StringProperty(name = "", description="The string value of the property")
    valueReference: bpy.props.PointerProperty(name = "", type=bpy.types.Object, description="The object to reference")
    valueReferenceScene: bpy.props.PointerProperty(name = "", type=bpy.types.Scene, description="The scene to search for an object to reference; if this is blank, all objects from all scenes will appear in the object search field")
    valueColor: bpy.props.FloatVectorProperty(name = "", description="The color value of the property", subtype="COLOR", default=[1, 1, 1, 1], size=4, min=0, max=1)
    valueVector3D: bpy.props.FloatVectorProperty(name = "", description="The 3D vector value of the property", subtype="XYZ")

    valueFilepath: bpy.props.StringProperty(name = "", description="The filepath of the property", subtype="FILE_PATH", set=filepathSet, get=filepathGet)
    valueDirpath: bpy.props.StringProperty(name = "", description="The directory path of the property", subtype="DIR_PATH")
    # valueFilepathAbsolute
    # valueVector4D: bpy.props.FloatVectorProperty(name = "", description="The 4D vector value of the property")

class OBJECT_OT_tetra3dAddProp(bpy.types.Operator):
    bl_idname = "object.tetra3daddprop"
    bl_label = "Add Game Property"
    bl_description= "Adds a game property to the currently selected object. A game property gets added to an Object's Properties object in Tetra3D"
    bl_options = {'REGISTER', 'UNDO'}

    mode : bpy.props.StringProperty()

    def execute(self, context):
        
        if self.mode == "scene":
            target = context.scene
        elif self.mode == "object":
            target = context.object
        elif self.mode == "material":
            target = context.object.active_material
        elif self.mode == "action":
            target = context.active_action

        # target = getattr(context, self.mode)
        target.t3dGameProperties__.add()
        target.t3dGameProperties__.move(len(target.t3dGameProperties__)-1, 0)
        return {'FINISHED'}

class OBJECT_OT_tetra3dDeleteProp(bpy.types.Operator):
    bl_idname = "object.tetra3ddeleteprop"
    bl_label = "Delete Game Property"
    bl_description= "Deletes a game property from the currently selected object"
    bl_options = {'REGISTER', 'UNDO'}

    index : bpy.props.IntProperty()
    mode : bpy.props.StringProperty()

    def execute(self, context):

        if self.mode == "scene":
            target = context.scene
        elif self.mode == "object":
            target = context.object
        elif self.mode == "material":
            target = context.object.active_material
        elif self.mode == "action":
            target = context.active_action
           
        prop = target.t3dGameProperties__[self.index]

        global currentlyPlayingAudioHandle, currentlyPlayingAudioName

        if prop.valueType == "file" and prop.valueFilepath == currentlyPlayingAudioName and currentlyPlayingAudioHandle:
            currentlyPlayingAudioHandle.stop()
            currentlyPlayingAudioHandle = None

        target.t3dGameProperties__.remove(self.index)

        return {'FINISHED'}

class OBJECT_OT_tetra3dReorderProps(bpy.types.Operator):
    bl_idname = "object.tetra3dreorderprops"
    bl_label = "Re-order Game Property"
    bl_description= "Moves a game property up or down in the list"
    bl_options = {'REGISTER', 'UNDO'}

    index : bpy.props.IntProperty()
    moveUp : bpy.props.BoolProperty()
    mode : bpy.props.StringProperty()

    def execute(self, context):

        if self.mode == "scene":
            target = context.scene
        elif self.mode == "object":
            target = context.object
        elif self.mode == "material":
            target = context.object.active_material
        elif self.mode == "action":
            target = context.active_action

        if self.moveUp:
            target.t3dGameProperties__.move(self.index, self.index-1)
        else:
            target.t3dGameProperties__.move(self.index, self.index+1)

        return {'FINISHED'}

class OBJECT_OT_tetra3dSetVector(bpy.types.Operator):
    bl_idname = "object.t3dsetvec"
    bl_label = "" ## We don't want the label to show
    bl_description= "Sets vector value"
    bl_options = {'REGISTER', 'UNDO'}

    index : bpy.props.IntProperty()
    mode : bpy.props.StringProperty()
    buttonMode : bpy.props.StringProperty()

    @classmethod
    def description(cls, context, properties):
        if properties.buttonMode == "object location":
            return "Set to object world position"
        else:
            return "Set to 3D Cursor position"

    def execute(self, context):

        if self.mode == "scene":
            target = context.scene
        elif self.mode == "object":
            target = context.object
        elif self.mode == "material":
            target = context.object.active_material
        elif self.mode == "action":
            target = context.active_action

        if self.buttonMode == "object location":
            target.t3dGameProperties__[self.index].valueVector3D = context.object.location
        elif self.buttonMode == "3D cursor":
            target.t3dGameProperties__[self.index].valueVector3D = context.scene.cursor.location

        return {'FINISHED'}

class OBJECT_OT_tetra3dFocusObject(bpy.types.Operator):

    bl_idname = "object.t3dfocusobject"
    bl_label = "" ## We don't want the label to show
    bl_description= "Focuses the camera on the specified object"
    bl_options = {'REGISTER', 'UNDO'}

    target : bpy.props.StringProperty()

    def execute(self, context):

        if self.target != "":

            old_selected = bpy.context.selected_objects.copy()

            ob = bpy.data.objects[self.target]
            bpy.ops.object.select_all(action='DESELECT')
            context.window.scene = ob.users_scene[0]
            context.window.view_layer = ob.users_scene[0].view_layers[0] 

            override_window = context.window
            override_screen = override_window.screen
            areas = [area for area in override_screen.areas if area.type == "VIEW_3D"]
            
            for area in areas:
                override_region = [region for region in area.regions if region.type == 'WINDOW']

                with context.temp_override(window=override_window, area=area, region=override_region[0]):
                    ob.select_set(True)
                    bpy.ops.view3d.view_selected()
                    context.region_data.view_distance += 25
                    # print(context.region_data)

            bpy.ops.object.select_all(action='DESELECT')
            for obj in old_selected:
                obj.select_set(True)

        return {'FINISHED'}

search_name = False

def enum_search(scene, context):

    options = set()

    def check_properties(o):
        global search_name
        for p in o.t3dGameProperties__:
            if search_name:
                options.add((p.name, p.name, ""))
            else:
                if p.valueType == "string":
                    options.add((p.valueString, p.valueString, ""))

    for o in bpy.data.objects:
        check_properties(o)
    for o in bpy.data.materials:
        check_properties(o)
    for o in bpy.data.actions:
        check_properties(o)
    for o in bpy.data.scenes:
        check_properties(o)

    result = sorted(list(options), key=lambda x: x[0].lower())
    
    return result

class OBJECT_OT_tetra3dSearchStringProperties(bpy.types.Operator):

    bl_idname = "object.t3dsearchstringproperties"
    bl_label = ""
    bl_description = "Search for previously-used strings in the blend file"
    bl_options = {'REGISTER', 'UNDO'}
    bl_property = "search_options"

    index : bpy.props.IntProperty()
    search_options: bpy.props.EnumProperty(name="Search Options", items=enum_search)
    mode : bpy.props.StringProperty()
    search_name : bpy.props.BoolProperty()

    def execute(self, context):

        if self.mode == "scene":
            target = context.scene
        elif self.mode == "object":
            target = context.object
        elif self.mode == "material":
            target = context.object.active_material
        elif self.mode == "action":
            target = context.active_action

        if self.search_name:
            target.t3dGameProperties__[self.index].name = self.search_options
        else:
            target.t3dGameProperties__[self.index].valueString = self.search_options

        return {'FINISHED'}

    def invoke(self, context, event):
        global search_name
        if self.search_name:
            search_name = True
        else:
            search_name = False
        context.window_manager.invoke_search_popup(self)
        return {'FINISHED'}

def copyProp(fromProp, toProp):
    toProp.name = fromProp.name
    toProp.valueType = fromProp.valueType
    toProp.valueBool = fromProp.valueBool
    toProp.valueInt = fromProp.valueInt
    toProp.valueFloat = fromProp.valueFloat
    toProp.valueString = fromProp.valueString
    toProp.valueReference = fromProp.valueReference
    toProp.valueReferenceScene = fromProp.valueReferenceScene
    toProp.valueColor = fromProp.valueColor
    toProp.valueVector3D = fromProp.valueVector3D
    toProp.valueFilepath = fromProp.valueFilepath
    toProp.valueDirpath = fromProp.valueDirpath

class OBJECT_OT_tetra3dOverrideProp(bpy.types.Operator):
    bl_idname = "object.tetra3doverrideprop"
    bl_label = "Override Game Property"
    bl_description= "Copies a game property to the collection instance for overriding."
    bl_options = {'REGISTER', 'UNDO'}

    objectIndex : bpy.props.IntProperty()
    propIndex : bpy.props.IntProperty()

    def execute(self, context):

        targetProp = context.object.instance_collection.objects[self.objectIndex].t3dGameProperties__[self.propIndex]

        newProp = None

        for prop in context.object.t3dGameProperties__:
            if prop.name == targetProp.name:
                newProp = prop
                break

        if newProp is None:
            newProp = context.object.t3dGameProperties__.add()

        copyProp(targetProp, newProp)

        return {'FINISHED'}


class OBJECT_OT_tetra3dCopyProps(bpy.types.Operator):
    bl_idname = "object.tetra3dcopyprops"
    bl_label = "Copy Game Properties"
    bl_description= "Copies game properties from the currently selected object to all other selected objects"
    bl_options = {'REGISTER', 'UNDO'}

    def execute(self, context):

        selected = context.object

        for o in context.selected_objects:
            if o == selected:
                continue
            o.t3dGameProperties__.clear()
            for prop in selected.t3dGameProperties__:
                newProp = o.t3dGameProperties__.add()
                copyProp(prop, newProp)

        return {'FINISHED'}

class MATERIAL_OT_tetra3dMaterialCopyProps(bpy.types.Operator):
    bl_idname = "material.tetra3dcopyprops"
    bl_label = "Overwrite Game Property on All Materials"
    bl_description= "Overwrites game properties from the currently selected material to all other materials on this object"
    bl_options = {'REGISTER', 'UNDO'}

    def execute(self, context):

        selected = context.object

        for slot in selected.material_slots:
            if slot.material == None or slot.material == selected.active_material:
                continue
            slot.material.t3dGameProperties__.clear()
            for prop in selected.active_material.t3dGameProperties__:
                newProp = slot.material.t3dGameProperties__.add()
                copyProp(prop, newProp)

        return {'FINISHED'}

class OBJECT_OT_tetra3dCopyOneProperty(bpy.types.Operator):
    bl_idname = "object.tetra3dcopyoneproperty"
    bl_label = "Copy Game Property"
    bl_description= "Copies a single game property from the currently selected object to all other selected objects"
    bl_options = {'REGISTER', 'UNDO'}

    index : bpy.props.IntProperty()

    def execute(self, context):

        selected = context.object

        for o in context.selected_objects:
            if o == selected:
                continue
            
            fromProp = selected.t3dGameProperties__[self.index]

            if fromProp.name in o.t3dGameProperties__:
                toProp = o.t3dGameProperties__[fromProp.name]
            else:
                toProp = o.t3dGameProperties__.add()

            copyProp(fromProp, toProp)

        return {'FINISHED'}

class OBJECT_OT_tetra3dCopyNodePathToClipboard(bpy.types.Operator):
    bl_idname = "object.tetra3dcopynodepath"
    bl_label = "Copy Node Path To Clipboard"
    bl_description= "Copies an object's node path to clipboard"
    bl_options = {'REGISTER', 'UNDO'}

    def execute(self, context):
        bpy.context.window_manager.clipboard = objectNodePath(context.object)
        return {'FINISHED'}

class OBJECT_OT_tetra3dClearProps(bpy.types.Operator):
    bl_idname = "object.tetra3dclearprops"
    bl_label = "Clear Game Properties"
    bl_description= "Clears game properties from all currently selected objects"
    bl_options = {'REGISTER', 'UNDO'}

    mode : bpy.props.StringProperty()

    def execute(self, context):
        
        if self.mode == "object":

            for o in context.selected_objects:
                o.t3dGameProperties__.clear()

        elif self.mode == "scene":
            context.scene.t3dGameProperties__.clear()

        elif self.mode == "material" and context.object.active_material is not None:
            context.object.active_material.t3dGameProperties__.clear()

        return {'FINISHED'}

class OBJECT_OT_tetra3dPlaySample(bpy.types.Operator):

    bl_idname = "object.t3dplaysound"
    bl_label = "Preview Music File"
    bl_description= "Previews music file"
    bl_options = {'REGISTER'}

    filepath : bpy.props.StringProperty()

    def execute(self, context):
        
        global currentlyPlayingAudioHandle, currentlyPlayingAudioName, audioPaused

        device = aud.Device()
        
        if currentlyPlayingAudioHandle:
            if currentlyPlayingAudioName == self.filepath:
                currentlyPlayingAudioHandle.resume()
                audioPaused = False
            else:
                currentlyPlayingAudioHandle.stop()
        
        if not currentlyPlayingAudioHandle or currentlyPlayingAudioName != self.filepath:

            sound = aud.Sound(bpy.path.abspath(self.filepath))

            currentlyPlayingAudioHandle = device.play(sound)
            currentlyPlayingAudioHandle.volume = 0.5
            currentlyPlayingAudioHandle.loop_count = -1
            currentlyPlayingAudioName = self.filepath

        return {'FINISHED'}
    
class OBJECT_OT_tetra3dStopSample(bpy.types.Operator):

    bl_idname = "object.t3dpausesound"
    bl_label = "Pauses Previewing Music File"
    bl_description= "Stops currently playing music file"
    bl_options = {'REGISTER', 'UNDO'}

    filepath : bpy.props.StringProperty()

    def execute(self, context):
        
        global currentlyPlayingAudioHandle, audioPaused

        if currentlyPlayingAudioHandle:
            currentlyPlayingAudioHandle.pause()
            audioPaused = True

        return {'FINISHED'}

class OBJECT_OT_tetra3dSetAnimationInterpolationAll(bpy.types.Operator):

    bl_idname = "object.t3dsetanimationinterpolationall"
    bl_label = "All Animations"
    bl_description= "Sets interpolation for all keys in all animations in the blend file to the specified interpolation mode"
    bl_options = {'REGISTER'}

    interpolationType : bpy.props.StringProperty()

    def execute(self, context):

        for action in bpy.data.actions:
            for curve in action.fcurves:
                for point in curve.keyframe_points:
                    point.interpolation = self.interpolationType

        return {'FINISHED'}


class OBJECT_OT_tetra3dSetAnimationInterpolation(bpy.types.Operator):

    bl_idname = "object.t3dsetanimationinterpolation"
    bl_label = "Current Animation"
    bl_description= "Sets interpolation for all keys for the current animation on the object to the specified interpolation mode"
    bl_options = {'REGISTER'}

    interpolationType : bpy.props.StringProperty()
    forAll : bpy.props.BoolProperty()

    def execute(self, context):

        if bpy.context.active_object and bpy.context.active_object.animation_data and bpy.context.active_object.animation_data.action:
            action = bpy.context.active_object.animation_data.action
            for curve in action.fcurves:
                for point in curve.keyframe_points:
                    point.interpolation = self.interpolationType

        return {'FINISHED'}

class RENDER_OT_tetra3dQuickSetRenderResolution(bpy.types.Operator):

    bl_idname = "render.t3dquicksetrenderresolution"
    bl_label = "Set render resolution"
    bl_description= "Sets the render resolution for cameras in this blend file to quick-values"
    bl_options = {'REGISTER', 'UNDO'}

    resolutionHeight : bpy.props.IntProperty()

    def execute(self, context):

        asr = 16/9
        context.scene.t3dRenderResolutionW__ = int(self.resolutionHeight * asr)
        context.scene.t3dRenderResolutionH__ = int(self.resolutionHeight)
        return {'FINISHED'}

def objectNodePath(object):

    p = object.name

    if object.parent:

        if object.parent_type == "BONE":
            armaturePath = ""

            parentBoneName = object.parent_bone
            armatureData = object.parent.data
            parent = armatureData.bones[parentBoneName]

            while parent:
                armaturePath = parent.name + "/" + armaturePath
                parent = parent.parent

            p = objectNodePath(object.parent) + "/" + armaturePath + object.name

        else:
            p = objectNodePath(object.parent) + "/" + object.name

    return p

class MESH_PT_tetra3d(bpy.types.Panel):
    bl_idname = "MESH_PT_tetra3d"
    bl_label = "Tetra3d Mesh Properties"
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_context = "data"

    def draw(self, context): 

        row = self.layout.row()
        meshData = context.object.data
        if meshData and context.object.type == "MESH":
            row.prop(meshData, "t3dUniqueMesh__")
            
            row = self.layout.row()
            row.prop(meshData, "t3dUniqueMaterials__")
            if "t3dUniqueMesh__" in meshData:
                row.enabled = meshData["t3dUniqueMesh__"]
            else:
                row.enabled = False

class ACTION_PT_tetra3d(bpy.types.Panel):

    bl_idname = "ACTION_PT_tetra3d"
    bl_label = "Tetra3d Action Properties"
    bl_space_type = 'DOPESHEET_EDITOR'
    bl_region_type = 'UI'
    bl_context = "ANIMATION"
    bl_category = "Action"

    @classmethod
    def poll(self,context):
        return context.active_action is not None

    def draw(self, context):

        row = self.layout.row()
        row.prop(context.active_action, "t3dRelativeMotion__")
        
        row = self.layout.row()

        add = row.operator("object.tetra3daddprop", text="Add Game Property", icon="PLUS")
        add.mode = "action"

        # row.operator("object.tetra3dcopyprops", text="Overwrite All Game Properties", icon="COPYDOWN")

        handleT3DProperties(self, None, context.active_action.t3dGameProperties__, "action")

        row = self.layout.row()

        # No scene equivalent for this, so there is no mode property for this class
        clear = row.operator("object.tetra3dclearprops", text="Clear All Game Properties", icon="CANCEL")
        clear.mode = "action"

class OBJECT_PT_tetra3d(bpy.types.Panel):
    bl_idname = "OBJECT_PT_tetra3d"
    bl_label = "Tetra3d Object Properties"
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_context = "object"

    @classmethod
    def poll(self,context):
        return context.object is not None

    def draw(self, context):

        row = self.layout.row()
        np = objectNodePath(context.object)
        np = " / ".join(np.split("/"))
        row.label(text="Node Path : " + np)
        row = self.layout.row()
        row.operator("object.tetra3dcopynodepath", text="Copy Node Path to Clipboard", icon="COPYDOWN")
        
        row = self.layout.row()
        row.enabled = context.object.t3dObjectType__ == 'MESH'
        row.prop(context.object, "t3dVisible__")

        if context.object.type == "MESH":
            box = self.layout.box()
            row = box.row()
            row.enabled = context.object.t3dObjectType__ == 'MESH'
            row.prop(context.object, "t3dAutoBatch__")
            row = box.row()
            row.label(text="Object Type: ")
            row.prop(context.object, "t3dObjectType__", expand=True)

            row = box.row()
            row.enabled = context.object.t3dObjectType__ == 'MESH'
            row.prop(context.object, "t3dAutoSubdivide__")
            if context.object.t3dAutoSubdivide__:
                row.prop(context.object, "t3dAutoSubdivideSize__") 

        isCollection = context.object.instance_type == "COLLECTION" and context.object.instance_collection is not None

        box = self.layout.box()
        row = box.row()
        row.label(text="Sector Type:")
        
        if isCollection:
            row = box.row()
            row.enabled = context.object.t3dObjectType__ == 'MESH'
            row.prop(context.object, "t3dSectorTypeOverride__", expand=True)

        row = box.row()
        if isCollection:
            row.enabled = context.object.t3dSectorTypeOverride__
        else:
            row.enabled = context.object.t3dObjectType__ == 'MESH'
        row.prop(context.object, "t3dSectorType__", expand=True)

        row = self.layout.row()
        row.prop(context.object, "t3dBoundsType__")
        
        row = self.layout.row()
        
        if context.object.t3dBoundsType__ == 'AABB':
            row.prop(context.object, "t3dAABBCustomEnabled__")
            if context.object.t3dAABBCustomEnabled__:
                row = self.layout.row()
                row.prop(context.object, "t3dAABBCustomSize__")
        elif context.object.t3dBoundsType__ == 'CAPSULE':
            row.prop(context.object, "t3dCapsuleCustomEnabled__")
            if context.object.t3dCapsuleCustomEnabled__:
                row = self.layout.row()
                row.prop(context.object, "t3dCapsuleCustomRadius__")
                row.prop(context.object, "t3dCapsuleCustomHeight__")
        elif context.object.t3dBoundsType__ == 'SPHERE':
            row.prop(context.object, "t3dSphereCustomEnabled__")
            if context.object.t3dSphereCustomEnabled__:
                row = self.layout.row()
                row.prop(context.object, "t3dSphereCustomRadius__")
        elif context.object.t3dBoundsType__ == 'TRIANGLES':
            row.prop(context.object, "t3dTrianglesCustomBroadphaseEnabled__")
            if context.object.t3dTrianglesCustomBroadphaseEnabled__:
                row.prop(context.object, "t3dTrianglesCustomBroadphaseGridSize__")

        row = self.layout.row()
        row.separator()

        if isCollection:

            row = self.layout.row()
            row.label(text="Collection Object Properties")
            row.prop(context.scene, "t3dExpandOverrideProps__", icon="TRIA_DOWN" if context.scene.t3dExpandOverrideProps__ else "TRIA_RIGHT", icon_only=True, emboss=False)

            if context.scene.t3dExpandOverrideProps__:

                col = context.object.instance_collection

                for objectIndex, object in enumerate(col.objects):

                    if object.parent == None:

                        row = self.layout.row()
                        box = row.box()
                        box.label(text="Object: " + object.name)
                        box.row().separator()
                            
                        handleT3DProperties(self, box, object.t3dGameProperties__, "object", False)

        row = self.layout.row()
        row.label(text="Game Properties")
        row.prop(context.scene, "t3dExpandGameProps__", icon="TRIA_DOWN" if context.scene.t3dExpandGameProps__ else "TRIA_RIGHT", icon_only=True, emboss=False)


        if context.scene.t3dExpandGameProps__:

            row = self.layout.row()

            add = row.operator("object.tetra3daddprop", text="Add Game Property", icon="PLUS")
            add.mode = "object"

            row.operator("object.tetra3dcopyprops", text="Overwrite All Game Properties", icon="COPYDOWN")

            handleT3DProperties(self, None, context.active_object.t3dGameProperties__, "object")

            row = self.layout.row()

            # No scene equivalent for this, so there is no mode property for this class
            clear = row.operator("object.tetra3dclearprops", text="Clear All Game Properties", icon="CANCEL")
            clear.mode = "object"


class SCENE_PT_tetra3d(bpy.types.Panel):
    bl_idname = "SCENE_PT_tetra3d"
    bl_label = "Tetra3d Scene Properties"
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_context = "scene"

    @classmethod
    def poll(self,context):
        return context.scene is not None

    def draw(self, context):

        row = self.layout.row()
        add = row.operator("object.tetra3daddprop", text="Add Game Property", icon="PLUS")
        add.mode = "scene"

        handleT3DProperties(self, None, context.scene.t3dGameProperties__, "scene")
        
        row = self.layout.row()
        clear = row.operator("object.tetra3dclearprops", text="Clear All Game Properties", icon="CANCEL")
        clear.mode = "scene"

class CAMERA_PT_tetra3d(bpy.types.Panel):
    bl_idname = "CAMERA_PT_tetra3d"
    bl_label = "Tetra3d Camera Properties"
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_context = "data"

    @classmethod
    def poll(self,context):
        return context.object is not None and context.object.type == "CAMERA"

    def draw(self, context):

        row = self.layout.row()
        row.prop(context.object.data, "type")
        row = self.layout.row()
        if context.object.data.type == "PERSP":
            row.prop(context.object.data, "t3dFOV__")
        else:
            row.prop(context.object.data, "ortho_scale")
        row = self.layout.row()
        row.prop(context.object.data, "clip_start")
        row.prop(context.object.data, "clip_end")
        row = self.layout.row()
        row.prop(context.object.data, "t3dMaxLightCount__")

        box = self.layout.box()
        box.prop(context.object.data, "t3dSectorRendering__")
        row = box.row()
        sectorRenderingOn = context.object.data.t3dSectorRendering__
        row.enabled = sectorRenderingOn
        row.prop(context.object.data, "t3dSectorRenderDepth__")
        row.enabled = sectorRenderingOn

        box = self.layout.box()
        row = box.row()
        row.prop(context.object.data, "t3dPerspectiveCorrectedTextureMapping__")


def handleT3DProperties(self, box, props, operatorType, enabled=True):

    for index, prop in enumerate(props):

        if box is None:
            box = self.layout.box()

        row = box.row()
        row.prop(prop, "name")

        op = row.operator(OBJECT_OT_tetra3dSearchStringProperties.bl_idname, text="", icon="VIEWZOOM")
        op.search_name = True
        op.index = index
        op.mode = operatorType
        
        moveUpOptions = row.operator(OBJECT_OT_tetra3dReorderProps.bl_idname, text="", icon="TRIA_UP")
        moveUpOptions.index = index
        moveUpOptions.moveUp = True
        moveUpOptions.mode = operatorType

        moveDownOptions = row.operator(OBJECT_OT_tetra3dReorderProps.bl_idname, text="", icon="TRIA_DOWN")
        moveDownOptions.index = index
        moveDownOptions.moveUp = False
        moveDownOptions.mode = operatorType

        copy = row.operator(OBJECT_OT_tetra3dCopyOneProperty.bl_idname, text="", icon="COPYDOWN")
        copy.index = index

        deleteOptions = row.operator(OBJECT_OT_tetra3dDeleteProp.bl_idname, text="", icon="TRASH")
        deleteOptions.index = index
        deleteOptions.mode = operatorType
        
        row = box.row()
        row.enabled = enabled
        row.prop(prop, "valueType")
        
        if prop.valueType == "bool":
            row.prop(prop, "valueBool")
        elif prop.valueType == "int":
            row.prop(prop, "valueInt")
        elif prop.valueType == "float":
            row.prop(prop, "valueFloat")
        elif prop.valueType == "string":
            row.prop(prop, "valueString")
            op = row.operator(OBJECT_OT_tetra3dSearchStringProperties.bl_idname, text="", icon="VIEWZOOM")
            op.index = index
            op.mode = operatorType
            op.search_name = False
        elif prop.valueType == "reference":
            row.prop(prop, "valueReferenceScene")
            if prop.valueReferenceScene != None:
                row.prop_search(prop, "valueReference", prop.valueReferenceScene, "objects")
            else:
                row.prop(prop, "valueReference")
            op = row.operator("object.t3dfocusobject", text="", icon="CAMERA_DATA")
            if prop.valueReference:
                op.target = prop.valueReference.name
        elif prop.valueType == "color":
            row.prop(prop, "valueColor")
        elif prop.valueType == "vector3d":
            row = box.row()
            row.enabled = enabled
            row.prop(prop, "valueVector3D")

            if operatorType == "object" or operatorType == "material":
                
                setCur = row.operator("object.t3dsetvec", text="", icon="OBJECT_ORIGIN")
                setCur.index = index
                setCur.mode = operatorType
                setCur.buttonMode = "object location"

            setCur = row.operator("object.t3dsetvec", text="", icon="PIVOT_CURSOR")
            setCur.index = index
            setCur.mode = operatorType
            setCur.buttonMode = "3D cursor"
        elif prop.valueType == "file":
            row.prop(prop, "valueFilepath")
            ext = os.path.splitext(prop.valueFilepath)[1]

            if ext in bpy.path.extensions_audio:
                global currentlyPlayingAudioHandle, audioPaused

                if currentlyPlayingAudioHandle and not audioPaused:
                    playButton = row.operator("object.t3dpausesound", text="", icon="PAUSE")
                else:
                    playButton = row.operator("object.t3dplaysound", text="", icon="PLAY")
                    playButton.filepath = prop.valueFilepath
        elif prop.valueType == "directory":
            row.prop(prop, "valueDirpath")
        
class MATERIAL_PT_tetra3d(bpy.types.Panel):
    bl_idname = "MATERIAL_PT_tetra3d"
    bl_label = "Tetra3d Material Properties"
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_context = "material"

    @classmethod
    def poll(self,context):
        return context.material is not None

    def draw(self, context):
        row = self.layout.row()
        row.prop(context.material, "t3dMaterialColor__")
        # row = self.layout.row()
        # row.prop(context.material, "t3dColorTexture0__")
        # row.operator("image.open")
        row = self.layout.row()
        row.prop(context.material, "t3dMaterialShadeless__")
        row.prop(context.material, "t3dMaterialFogless__")
        row = self.layout.row()
        row.prop(context.material, "use_backface_culling")
        row.prop(context.material, "t3dVisible__")
        row = self.layout.row()
        row.label(text="Transparency Mode:")
        row.prop(context.material, "t3dTransparencyMode__", text="")
        row = self.layout.row()
        row.label(text="Blend Mode:")
        row.prop(context.material, "t3dBlendMode__", text="")
        row = self.layout.row()
        row.label(text="Billboard Mode:")
        row.prop(context.material, "t3dBillboardMode__", text="")

        box = self.layout.box()
        row = box.row()
        row.prop(context.material, "t3dAutoUV__")
        row = box.row()
        row.enabled = context.material.t3dAutoUV__
        row.prop(context.material, "t3dAutoUVUnitSize__")
        row.prop(context.material, "t3dAutoUVRotation__")
        row = box.row()
        row.enabled = context.material.t3dAutoUV__
        row.prop(context.material, "t3dAutoUVOffset__")

        box = self.layout.box()
        row = box.row()
        row.prop(context.material, "t3dCustomDepthOn__")
        row = box.row()
        row.enabled = context.material.t3dCustomDepthOn__
        row.prop(context.material, "t3dCustomDepthValue__")
        row = box.row()
        row.label(text="Lighting Mode:")
        row.prop(context.material, "t3dMaterialLightingMode__", text="")

        if context.object.active_material != None:

            row = self.layout.row()
            add = row.operator("object.tetra3daddprop", text="Add Game Property", icon="PLUS")
            add.mode = "material"

            row.operator("material.tetra3dcopyprops", icon="COPYDOWN")
            
            handleT3DProperties(self, None, context.object.active_material.t3dGameProperties__, "material")

            row = self.layout.row()
            clear = row.operator("object.tetra3dclearprops", text="Clear All Game Properties", icon="CANCEL")
            clear.mode = "material"

class WORLD_PT_tetra3d(bpy.types.Panel):
    bl_idname = "WORLD_PT_tetra3d"
    bl_label = "Tetra3d World Properties"
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_context = "world"

    @classmethod
    def poll(self,context):
        return context.world is not None

    def draw(self, context):
        row = self.layout.row()
        row.prop(context.world, "t3dSyncClearColor__")
        row = self.layout.row()
        row.prop(context.world, "t3dClearColor__")
        row.enabled = not context.world.t3dSyncClearColor__
        box = self.layout.box()
        row = box.row()
        row.prop(context.world, "t3dFogMode__")

        if context.world.t3dFogMode__ != "OFF":

            box.prop(context.world, "t3dFogCurve__")

            box.prop(context.world, "t3dSyncFogColor__")

            row = box.row()
            row.prop(context.world, "t3dFogColor__")
            row.enabled = context.world.t3dFogMode__ != "TRANSPARENT" and not context.world.t3dSyncFogColor__

            box.prop(context.world, "t3dFogDithered__")            
            box.prop(context.world, "t3dFogRangeStart__", slider=True)
            box.prop(context.world, "t3dFogRangeEnd__", slider=True)
        
# The idea behind "globalget and set" is that we're setting properties on the first scene (which must exist), and getting any property just returns the first one from that scene
def globalGet(propName, default=None):
    if propName not in bpy.data.scenes[0] and default is not None:
        bpy.data.scenes[0][propName] = default
        
    return bpy.data.scenes[0][propName]

def globalSet(propName, value):
    bpy.data.scenes[0][propName] = value

def globalDel(propName):
    del bpy.data.scenes[0][propName]

class RENDER_PT_tetra3d(bpy.types.Panel):
    bl_idname = "RENDER_PT_tetra3d"
    bl_label = "Tetra3D Render Properties"
    bl_space_type = "PROPERTIES"
    bl_region_type = "WINDOW"
    bl_context = "render"
    
    def draw(self, context):

        row = self.layout.row()
        row.operator(EXPORT_OT_tetra3d.bl_idname)
        row = self.layout.row()
        row.prop(context.scene, "t3dExportOnSave__")

        row = self.layout.row()
        row.prop(context.scene, "t3dExportFilepath__")
        
        row = self.layout.row()
        row.prop(context.scene, "t3dExportFormat__")
        
        box = self.layout.box()
        row = box.row()
        row.active = context.scene.t3dExportFormat__ == "GLB"
        row.prop(context.scene, "t3dPackTextures__")

        box.prop(context.scene, "t3dExportCameras__")
        box.prop(context.scene, "t3dExportLights__")
        box.prop(context.scene, "t3dRenameInstancedObjects__")

        box = self.layout.box()

        row = box.row()
        row.label(text="Quick set render resolution:")


        row = box.row()
        row.label(text="PS1-like:")
        op = row.operator(RENDER_OT_tetra3dQuickSetRenderResolution.bl_idname, text="224p")
        op.resolutionHeight = 224

        op = row.operator(RENDER_OT_tetra3dQuickSetRenderResolution.bl_idname, text="480p")
        op.resolutionHeight = 480

        row = box.row()
        row.label(text="Other:")

        op = row.operator(RENDER_OT_tetra3dQuickSetRenderResolution.bl_idname, text="720p")
        op.resolutionHeight = 720

        op = row.operator(RENDER_OT_tetra3dQuickSetRenderResolution.bl_idname, text="1080p")
        op.resolutionHeight = 1080

        row = box.row()
        row.prop(context.scene, "t3dRenderResolutionW__")
        row.prop(context.scene, "t3dRenderResolutionH__")
        
        box = self.layout.box()
        row = box.row()
        row.label(text="Sector detection type:")

        row = box.row()
        row.prop(context.scene, "t3dSectorDetectionType__", expand=True)

        box = self.layout.box()
        row = box.row()
        row.label(text="Animation Playback Framerate (in Blender):")
        row = box.row()
        row.prop(context.scene, "t3dPlaybackFPS__")
        row = box.row().separator()
        row = box.row()
        row.prop(context.scene, "t3dAnimationSampling__")
        row = box.row()
        row.label(text="Set interpolation for animations:")
        row = box.row()
        row.prop(context.scene, "t3dAnimationInterpolation__")
        row = box.row()
        op = row.operator(OBJECT_OT_tetra3dSetAnimationInterpolationAll.bl_idname)
        op.interpolationType = context.scene.t3dAnimationInterpolation__
        op = row.operator(OBJECT_OT_tetra3dSetAnimationInterpolation.bl_idname)
        op.interpolationType = context.scene.t3dAnimationInterpolation__


def export():
    scene = bpy.context.scene

    # Loop through all objects, see if there are any that are in a collection, but are deleted; if so, remove them from the collections.

    for o in bpy.data.objects:
        if len(o.users_scene) == 0: # If an object is in a collection but not a scene, we remove it.
            bpy.data.objects.remove(o)

    was_edit_mode = False
    old_active = bpy.context.active_object
    old_selected = bpy.context.selected_objects.copy()
    if bpy.context.mode == 'EDIT_MESH':
        bpy.ops.object.mode_set(mode='OBJECT')
        was_edit_mode = True
        
    blendPath = bpy.context.blend_data.filepath
    if scene.t3dExportFilepath__ != "":
        blendPath = scene.t3dExportFilepath__

    if blendPath == "":
        return False

    blendPath = bpy.path.abspath(blendPath)
    
    if scene.t3dExportFormat__ == "GLB":
        ending = ".glb"
    elif scene.t3dExportFormat__ == "GLTF_SEPARATE":
        ending = ".gltf"
    
    newPath = os.path.splitext(blendPath)[0] + ending

    # Gather collection information
    ogCollections = {} # What collection an object was originally pointing to
    collections = {} # What collections exist in the Blend file
    ogGrids = {}

    for collection in bpy.data.collections:
        if len(collection.objects) == 0:
            continue
        c = []
        for o in collection.objects:
            if o.parent is None:
                c.append(o.name)

        cd = {
            "objects": c,
            "offset" : collection.instance_offset,
        }

        if collection.library is not None:
            cd["path"] = collection.library.filepath

        collections[collection.name] = cd
    
    globalSet("t3dCollections__", collections)

    worlds = {}

    for world in bpy.data.worlds:

        worldData = {}

        worldNodes = world.node_tree.nodes
        
        # If you're using nodes, it'll try to use either a background or emission node; otherwise, it'll just use the background color
        if ("Background" in worldNodes or "Emission" in worldNodes) and world.use_nodes:
            if "Background"in worldNodes:
                bgNode = worldNodes["Background"]
            else:
                bgNode = worldNodes["Emission"]
            worldData["ambient color"] = list(bgNode.inputs[0].default_value)
            worldData["ambient energy"] = bgNode.inputs[1].default_value
        else:
            worldData["ambient color"] = list(world.color)
            worldData["ambient energy"] = 1

        if world.t3dSyncClearColor__:
            worldData["clear color"] = worldData["ambient color"]
        else:
            worldData["clear color"] = world.t3dClearColor__

        worldData["fog mode"] = world.t3dFogMode__
        worldData["dithered transparency"] = world.t3dFogDithered__
        worldData["fog curve"] = world.t3dFogCurve__

        if world.t3dSyncFogColor__:
            worldData["fog color"] = worldData["clear color"]
        else:
            worldData["fog color"] = world.t3dFogColor__

        worldData["fog range start"] = world.t3dFogRangeStart__
        worldData["fog range end"] = world.t3dFogRangeEnd__

        worlds[world.name] = worldData

    globalSet("t3dWorlds__", worlds)

    currentFrame = {}

    autoSubdivides = {}

    ogVertexColorNames = {}

    referencedActions = {}

    def checkAction(action):
        if obj.type == "ARMATURE":
            if action.name not in referencedActions:
                referencedActions[action.name] = set()
            referencedActions[action.name].add(obj.data.name)

    for scene in bpy.data.scenes:

        currentFrame[scene] = scene.frame_current

        if scene.users > 0:

            if scene.world:
                scene["t3dCurrentWorld__"] = scene.world.name

            for layer in scene.view_layers:
                for obj in layer.objects:
                    if obj.animation_data:
                        if obj.animation_data.action:
                            checkAction(obj.animation_data.action)
                        for track in obj.animation_data.nla_tracks.values():
                            for strip in track.strips.values():
                                checkAction(strip.action)

                    obj["t3dOriginalLocalPosition__"] = obj.location

                    if obj.type == "MESH":

                        # BUG: This causes a problem when subdividing; this is only really a problem if automatic tesselation when rendering in Tetra3D isn't implemented, though

                        if obj.t3dAutoSubdivide__:

                            obj["t3dOriginalMesh"] = obj.data.name

                            if not obj.data.name in autoSubdivides:

                                autoSubdivides[obj.data.name] = {
                                    "edit": obj.data,
                                    "original": obj.data.copy(),
                                    "size": obj.t3dAutoSubdivideSize__,
                                }
                        
                        if len(obj.vertex_groups) > 0:
                            vertexGroups = [group.name for group in obj.vertex_groups]
                            obj.data["t3dVertexGroupNames__"] = vertexGroups

                        # if len(obj.vertex_groups) > 0:
                        #     groupsByNames = {}
                        #     for group in obj.vertex_groups:
                        #         groupsByNames[group.name] = []
                        #         for i in range(vertexMaxCount):
                        #             if group.weight(i) > 0:
                        #                 groupsByNames[group.name].append(i)
                        #     obj.data["t3dVertexGroups__"] = vertexGroups
                            # vertexGroups = [group.name for group in obj.vertex_groups]
                            # obj.data["t3dVertexGroupNames__"] = vertexGroups

                        if len(obj.data.color_attributes) > 0:
                            vertexColors = [layer.name for layer in obj.data.color_attributes]
                            obj.data["t3dVertexColorNames__"] = vertexColors
                            obj.data["t3dActiveVertexColorIndex__"] = obj.data.color_attributes.render_color_index
                            ogVertexColorNames[obj.data] = list(layer.name for layer in obj.data.color_attributes)
                            
                            for layer in obj.data.color_attributes:
                                layer.name = "_"+layer.name # because the GLTF exporter now only exports color attributes if their layer names start with "_" for whatever reason~~~

                        if obj.t3dObjectType__ == 'GRID':
                            gridConnections = {}
                            gridEntries = []
                            ogGrids[obj] = obj.data
                            # obj.data = None # Hide the data just in case - that way Grid objects don't get mesh data exported
                            obj.data["t3dGrid__"] = True

                            for edge in obj.data.edges:
                                v0 = str(obj.data.vertices[edge.vertices[0]].co.to_tuple(4))
                                v1 = str(obj.data.vertices[edge.vertices[1]].co.to_tuple(4))

                                if v0 not in gridEntries:
                                    gridEntries.append(v0)
                                    gridConnections[str(gridEntries.index(v0))] = []
                                if v1 not in gridEntries:
                                    gridEntries.append(v1)
                                    gridConnections[str(gridEntries.index(v1))] = []

                                gridConnections[str(gridEntries.index(v0))].append(str(gridEntries.index(v1)))
                                gridConnections[str(gridEntries.index(v1))].append(str(gridEntries.index(v0)))
                                
                            obj["t3dGridConnections__"] = gridConnections
                            obj["t3dGridEntries__"] = gridEntries

                    # Record relevant information for curves
                    if obj.type == "CURVE":
                        points = []

                        for spline in obj.data.splines:
                            for point in spline.points:
                                points.append(point.co)
                            for point in spline.bezier_points:
                                points.append(point.co)

                        obj["t3dPathPoints__"] = points
                        obj["t3dPathCyclic__"] = spline.use_cyclic_u or spline.use_cyclic_v

                    if obj.instance_type == "COLLECTION" and obj.instance_collection is not None:
                        obj["t3dInstanceCollection__"] = obj.instance_collection.name
                        ogCollections[obj] = obj.instance_collection
                        # We don't want to export a linked collection directly, as that 1) will duplicate mesh data from externally linked blend files to put into the GLTF file, and
                        # 2) will apply the collection's offset to the object's position for some reason (which is annoying because we use OpenGL's axes for positioning compared to Blender)
                        obj.instance_collection = None

            for meshName in autoSubdivides:
        
                mesh = autoSubdivides[meshName]

                bm = bmesh.new()

                bm.from_mesh(mesh["edit"])
                bm.select_mode = {"EDGE", "VERT", "FACE"}

                # The below works, but the triangulation is super wonky and over-heavy

                # bmesh.ops.triangulate(bm, faces=bm.faces)

                # for x in range(1000):

                #     subdiv = False

                #     edges = []

                #     for edge in bm.edges:

                #         edge.select = edge.calc_length() > mesh["size"]
                #         if edge.select:
                #             subdiv = True
                #             edges.append(edge)
                    
                #     if not subdiv:
                #         break

                #     bm.select_flush(True)

                #     bmesh.ops.subdivide_edges(bm, edges=[e for e in bm.edges if e.select], cuts=1)

                #     bmesh.ops.triangulate(bm, faces=bm.faces)

                # The below works really well, but tris mess it up, I think

                invalidEdges = set()

                for x in range(100):

                    edges = set()

                    workingEdge = None

                    edgeCount = len(bm.edges)

                    for edge in bm.edges:

                        if edge in invalidEdges:
                            continue

                        if edge.calc_length() > mesh["size"]:

                            edges.add(edge)
                            if workingEdge is None and len(edge.link_loops) > 0:
                                workingEdge = edge

                            if len(edge.link_loops) > 0:

                                nextLoop = edge.link_loops[0]

                                passedCount = 0

                                for x in range(100):

                                    nextLoop = nextLoop.link_loop_next.link_loop_next.link_loop_radial_next
            
                                    if nextLoop.edge == workingEdge:
                                        passedCount += 1
                                        if passedCount >= 2:
                                            break

                                    edges.add(nextLoop.edge)

                        if workingEdge:
                            break

                    if len(edges) == 0:
                        break

                    bmesh.ops.subdivide_edgering(bm, edges=list(edges), cuts=1, profile_shape="LINEAR", smooth=0)

                    if len(bm.edges) == edgeCount:
                        invalidEdges.add(workingEdge)

                # Subdivide individual islands of faces that can't be loop-cut

                for x in range(100):

                    toCut = []

                    for edge in bm.edges:

                        if edge.calc_length() > mesh["size"]:

                            toCut.append(edge)

                    if len(toCut) == 0:
                        break

                    bmesh.ops.subdivide_edges(bm, edges=toCut, cuts=1)

                ################

                # for x in range(1):

                #     bm.faces.index_update()
                #     bm.edges.index_update()
                #     bm.verts.index_update()

                #     edges = set()

                #     for face in bm.faces:

                #         # subdivide non-quad faces later
                #         if len(face.edges) != 4:
                #             continue

                #         firstEdge = None

                #         for edge in face.edges:

                #             print(edge.calc_length())

                #             if edge.calc_length() > mesh["size"]:

                #                 firstEdge = edge
                #                 edges.add(edge)
                #                 break

                #         if firstEdge:

                #             nextLoop = edge.link_loops[0]

                #             for x in range(1000):

                #                 nextLoop = nextLoop.link_loop_next.link_loop_next.link_loop_radial_next

                #                 edges.add(nextLoop.edge)

                #             if len(edges) > 0:
                #                 break
                            
                #             if len(edges) == 0:
                #                 continue

                #     bmesh.ops.subdivide_edgering(bm, edges=list(edges), cuts=1, profile_shape="LINEAR", smooth=0)

                bm.to_mesh(mesh["edit"])

                bm.free()

    # Gather marker information and put them into the actions.
    for action in bpy.data.actions:
        markers = []
        for marker in action.pose_markers:
            markerInfo = {
                "name": marker.name,
                "time": marker.frame / globalGet("t3dPlaybackFPS__", 60), # If the playback FPS isn't specifically set, default to 60
            }
            markers.append(markerInfo)
        if len(markers) > 0:
            action["t3dMarkers__"] = markers

    view3DCameraData = []

    renderResolutionH = getRenderResolutionH(None)

    bpy.context.evaluated_depsgraph_get()

    for area in bpy.context.screen.areas:

        for space in area.spaces:

            if space.type == "VIEW_3D":

                # HUGE thanks to ryan halliday's blog for mentioning bpy_extras' axis conversion: https://blog.ryanhalliday.com/2023/04/three-blender-co-ordinates.html
                conversion = bpy_extras.io_utils.axis_conversion(from_forward="-Y", from_up="Z", to_forward="Z", to_up="Y")

                decomposed = space.region_3d.view_matrix.inverted().decompose()

                loc = conversion @ decomposed[0]
                rot = conversion @ (decomposed[1].to_matrix())

                camData = {
                    "clip_start" : space.clip_start,
                    "clip_end" : space.clip_end,
                    "location": loc,
                    "rotation" : rot,
                    "fovY" : math.degrees(2 * math.atan(36 / (space.lens * 2))), # 36 is the default blender camera sensor width
                    "perspective": space.region_3d.is_perspective,
                    # "ortho_zoom" : space.region_3d.view_distance, # This isn't correct; the lens also impacts the zoom
                }

                view3DCameraData.append(camData)

    globalSet("t3dView3DCameraData__", view3DCameraData)

    # We force on exporting of Extra values because otherwise, values from Blender would not be able to be exported.
    # export_apply=True to ensure modifiers are applied.
    bpy.ops.export_scene.gltf(
        filepath=newPath, 
        # use_active_scene=True, # Blender's GLTF exporter's kinda thrashed when it comes to multiple scenes, so it might be better to export each scene as its own GLTF file...?
        export_format=scene.t3dExportFormat__, 
        export_cameras=scene.t3dExportCameras__, 
        export_lights=scene.t3dExportLights__, 
        export_keep_originals=not scene.t3dPackTextures__,
        
        export_vertex_color='ACTIVE',
        export_attributes=True,
        
        export_current_frame=False,
        export_nla_strips=True,
        export_animations=True,
        export_frame_range=False,
        export_force_sampling=scene.t3dAnimationSampling__, # When enabled, animations are sampled / baked. This is slow, but accurate. When disabled, only linear and constant keyframes are exported and interpolated for animation.

        export_extras=True,
        export_yup=True,
        export_apply=True,
        export_import_convert_lighting_mode="COMPAT", # We want to use the compatible lighting model, not the "realistic" / real-world-accurate one
    )
    
    # Undo changes that we've made after export

    for meshName in autoSubdivides:

        mesh = autoSubdivides[meshName]

        bm = bmesh.new()
        bm.from_mesh(mesh["original"])
        bm.to_mesh(mesh["edit"])
        bm.free()

        removed = False

        try:
            mesh["original"].user_clear()
            removed = True
        except:
            pass

        if removed:
            try:
                bpy.data.meshes.remove(mesh["original"])
            except:
                pass

    for scene in bpy.data.scenes:

        # Exporting animations sets the frame "late"; we restore the current frame to avoid this
        scene.frame_set(currentFrame[scene])

        if scene.world and "t3dCurrentWorld__" in scene:
            del(scene["t3dCurrentWorld__"])

        if scene.users > 0:

            for layer in scene.view_layers:

                for obj in layer.objects:

                    if obj is None:
                        continue

                    if "t3dOriginalMesh" in obj:
                        del(obj["t3dOriginalMesh"])

                    if "t3dOriginalLocalPosition__" in obj:
                        del(obj["t3dOriginalLocalPosition__"])
                        
                    if "t3dInstanceCollection__" in obj:
                        del(obj["t3dInstanceCollection__"])
                        if obj in ogCollections:
                            obj.instance_collection = ogCollections[obj]
                    if "t3dPathPoints__" in obj:
                        del(obj["t3dPathPoints__"])
                    if "t3dPathCyclic__" in obj:
                        del(obj["t3dPathCyclic__"])
                    if obj.type == "MESH":
                        if "t3dVertexColorNames__" in obj.data:
                            del(obj.data["t3dVertexColorNames__"])
                        if "t3dActiveVertexColorIndex__" in obj.data:
                            del(obj.data["t3dActiveVertexColorIndex__"])
                        for i, layer in enumerate(obj.data.color_attributes):
                            layer.name = ogVertexColorNames[obj.data][i]
                        if obj.t3dObjectType__ == 'GRID':
                            del(obj.data["t3dGrid__"])
                            del(obj["t3dGridConnections__"])
                            del(obj["t3dGridEntries__"])
                            obj.data = ogGrids[obj] # Restore the mesh reference afterward


    for action in bpy.data.actions:
        if "t3dMarkers__" in action:
            del(action["t3dMarkers__"])

    globalDel("t3dView3DCameraData__")
    globalDel("t3dCollections__")
    globalDel("t3dWorlds__")

    # restore context
    bpy.ops.object.select_all(action='DESELECT')
    if old_active:
        old_active.select_set(True)
        bpy.context.view_layer.objects.active = old_active
    if bpy.context.active_object is not None and was_edit_mode:
        bpy.ops.object.mode_set(mode='EDIT')
    for obj in old_selected:
        if obj:
            obj.select_set(True)

    problematicActions = ""
    for actionName, armatureSet in referencedActions.items():

        if len(armatureSet) > 1:
            problematicActions += actionName + ", "

    if len(problematicActions) > 0:
        def report(self, context):
            self.layout.label(text="Warning: The following actions are assigned to or set in the NLA Stash for multiple different armatures: [ " + problematicActions + "]. This can be problematic if these armatures have different shapes, as the animations may be incorrect for the armature.")
        bpy.context.window_manager.popup_menu(report, title="Warning: Action Export Issue", icon="ERROR")

    return True

@persistent
def exportOnSave(dummy):
    
    if globalGet("t3dExportOnSave__", False):
        export()

@persistent
def onLoad(dummy):

    global currentlyPlayingAudioHandle, currentlyPlayingAudioName, audioPaused

    if currentlyPlayingAudioHandle:
        currentlyPlayingAudioHandle.stop()
        currentlyPlayingAudioHandle = None
        currentlyPlayingAudioName = ""
        audioPaused = False

    bpy.msgbus.subscribe_rna(
        key=(bpy.types.Object, 'mode'),
        owner="tetra3d",
        args=tuple(),
        notify=onModeChange,
    )

class MATERIAL_OT_tetra3dAutoUV(bpy.types.Operator):

    bl_idname = "material.t3dautouv"
    bl_label = "Apply Auto UV"
    bl_description = "Perform automatic UV mapping"
    bl_options = {'REGISTER', 'UNDO'}

    def execute(self, context):
        
        obj = bpy.context.object
        
        # If there's no polygons, we'll just return early
        if len(obj.data.polygons) == 0:
            return {'FINISHED'}
        
        ogMode = bpy.context.object.mode

        prevActiveMaterialIndex = obj.active_material_index

        # We need to be in Object mode to get vertex / edge / polygon selection data
        bpy.ops.object.mode_set(mode='OBJECT')
        selectedVertices = [v.index for v in obj.data.vertices if v.select]
        selectedEdges = [e.index for e in obj.data.edges if e.select]
        selectedPolygons = [p.index for p in obj.data.polygons if p.select]

        for matIndex, mat in enumerate(obj.data.materials):

            # mat could be None if it's just the slot and no Material is selected
            if mat and mat.t3dAutoUV__:

                bpy.ops.object.mode_set(mode='EDIT')

                bpy.ops.mesh.select_all(action='DESELECT')

                obj.active_material_index = matIndex
                bpy.ops.object.material_slot_select()

                # TODO: Replace cube_project with maybe manually stretching the UV values to be able to handle walls that have fixed widths or heights better (e.g. walls with trim)?
                bpy.ops.uv.cube_project(cube_size=mat.t3dAutoUVUnitSize__)

                bpy.ops.mesh.select_all(action='DESELECT')

                # Return to object mode to alter the UV map
                bpy.ops.object.mode_set(mode='OBJECT')

                uvMap = obj.data.uv_layers.active.uv

                for poly in obj.data.polygons:
                    if poly.material_index == matIndex:
                        for loopIndex in poly.loop_indices:
                            vec = uvMap[loopIndex].vector
                            vec.x -= 0.5
                            vec.y -= 0.5
                            vec.rotate(mathutils.Matrix.Rotation(math.radians(mat.t3dAutoUVRotation__), 2))
                            vec.x += 0.5
                            vec.y += 0.5

                            vec.x -= mat.t3dAutoUVOffset__[0]
                            vec.y -= mat.t3dAutoUVOffset__[1]

        bpy.ops.object.mode_set(mode='OBJECT')

        # Restore selection
        for v in obj.data.vertices:
            v.select = False
            if v.index in selectedVertices:
                v.select = True

        for e in obj.data.edges:
            e.select = False
            if e.index in selectedEdges:
                e.select = True

        for p in obj.data.polygons:
            p.select = False
            if p.index in selectedPolygons:
                p.select = True

        obj.active_material_index = prevActiveMaterialIndex

        bpy.ops.object.mode_set(mode=ogMode)

        return {'FINISHED'}

def onModeChange():
    obj = bpy.context.object
    if obj and obj.data and obj.type == 'MESH':
        for mat in obj.data.materials:
            if mat.t3dAutoUV__:
                bpy.ops.material.t3dautouv()
                break

def autoUVChange(self, context):
    obj = context.object
    if obj and obj.data and obj.type == 'MESH':
        for mat in obj.data.materials:
            if mat.t3dAutoUV__:
                bpy.ops.material.t3dautouv()
                break

class EXPORT_OT_tetra3d(bpy.types.Operator):
   bl_idname = "export.tetra3dgltf"
   bl_label = "Tetra3D Export"
   bl_description= "Exports to a GLTF file for use in Tetra3D"
   bl_options = {'REGISTER', 'UNDO'}

   def execute(self, context):
        if export():
            self.report({"INFO"}, "Tetra3D GLTF data exported properly.")
        else:
            self.report({"WARNING"}, "Warning: Tetra3D GLTF file could not be exported; please either specify a filepath or save the blend file.")
        return {'FINISHED'}


def getSectorDetectionType(self):
    return globalGet("t3dSectorDetection__", 0)

def setSectorDetectionType(self, value):
    globalSet("t3dSectorDetection__", value)

#####

def getRenderResolutionW(self):
    s = globalGet("t3dRenderResolutionW__", 640)
    bpy.context.scene.render.resolution_x = s
    return s

def setRenderResolutionW(self, value):
    globalSet("t3dRenderResolutionW__", value)
    bpy.context.scene.render.resolution_x = value

#####

def getRenderResolutionH(self):
    s = globalGet("t3dRenderResolutionH__", 360)
    bpy.context.scene.render.resolution_y = s
    return s

def setRenderResolutionH(self, value):
    globalSet("t3dRenderResolutionH__", value)
    bpy.context.scene.render.resolution_y = value

######

def getPlaybackFPS(self):
    s = globalGet("t3dPlaybackFPS__", 60)
    bpy.context.scene.render.fps = s
    return s

def setPlaybackFPS(self, value):
    globalSet("t3dPlaybackFPS__", value)
    bpy.context.scene.render.fps = value

# row = self.layout.row()
# row.prop(context.scene.render, "resolution_x")
# row.prop(context.scene.render, "resolution_y")
# row = self.layout.row()
# row.label(text="Animation Playback Framerate (in Blender):")
# row = self.layout.row()
# row.prop(context.scene.render, "fps")

def getExportOnSave(self):
    return globalGet("t3dExportOnSave__", False)

def setExportOnSave(self, value):
    globalSet("t3dExportOnSave__", value)



def getExportFilepath(self):
    return globalGet("t3dExportFilepath__", "")

def setExportFilepath(self, value):
    globalSet("t3dExportFilepath__", value)



def getExportFormat(self):
    return globalGet("t3dExportFormat__", 0)

def setExportFormat(self, value):
    globalSet("t3dExportFormat__", value)



def getExportCameras(self):
    return globalGet("t3dExportCameras__", True)

def setExportCameras(self, value):
    globalSet("t3dExportCameras__", value)



def getExportLights(self):
    return globalGet("t3dExportLights__", True)

def setExportLights(self, value):
    globalSet("t3dExportLights__", value)


def getPackTextures(self):
    return globalGet("t3dPackTextures__", False)

def setPackTextures(self, value):
    globalSet("t3dPackTextures__", value)

def getRenameInstancedObjects(self):
    return globalGet("t3dRenameInstancedObjects__", True)

def setRenameInstancedObjects(self, value):
    globalSet("t3dRenameInstancedObjects__", value)


def getAnimationSampling(self):
    return globalGet("t3dAnimationSampling__", True)

def setAnimationSampling(self, value):
    globalSet("t3dAnimationSampling__", value)

def getAnimationInterpolation(self):
    return globalGet("t3dAnimationInterpolation__", True)

def setAnimationInterpolation(self, value):
    globalSet("t3dAnimationInterpolation__", value)

def fogRangeStartSet(self, value):
    if value > bpy.context.world.t3dFogRangeEnd__:
        value = bpy.context.world.t3dFogRangeEnd__
    self["t3dFogRangeStart__"] = value

def fogRangeStartGet(self):
    if "t3dFogRangeStart__" in self:
        return self["t3dFogRangeStart__"]
    return 0

def fogRangeEndSet(self, value):
    if value < bpy.context.world.t3dFogRangeStart__:
        value = bpy.context.world.t3dFogRangeStart__
    self["t3dFogRangeEnd__"] = value

def fogRangeEndGet(self):
    if "t3dFogRangeEnd__" in self:
        return self["t3dFogRangeEnd__"]
    return 1

####

# We don't need to actually store a FOV value, but rather modify the Blender camera's usual FOV variable
def getFOV(self):

    # Huge thanks to this blender.stackexchange post: https://blender.stackexchange.com/questions/23431/how-to-set-camera-horizontal-and-vertical-fov

    w = getRenderResolutionW(None)
    h = getRenderResolutionH(None)
    aspect = w / h

    if aspect > 1:
        value = math.degrees(2 * math.atan((0.5 * h) / (0.5 * w / math.tan(self.angle / 2))))
    else:
        value = math.degrees(self.angle)

    return int(value)

def setFOV(self, value):

    w = getRenderResolutionW(None)
    h = getRenderResolutionH(None)
    aspect = w / h

    if aspect > 1:
        self.angle = 2 * math.atan((0.5 * w) / (0.5 * h / math.tan(math.radians(value) / 2)))
    else:
        self.angle = math.radians(value)

####

objectProps = {
    "t3dVisible__" : bpy.props.BoolProperty(name="Visible", description="Whether the object is visible or not when exported to Tetra3D", default=True),
    "t3dBoundsType__" : bpy.props.EnumProperty(items=boundsTypes, name="Bounds", description="What Bounding node type to create and parent to this object"),
    "t3dAABBCustomEnabled__" : bpy.props.BoolProperty(name="Custom AABB Size", description="If enabled, you can manually set the BoundingAABB node's size. If disabled, the AABB's size will be automatically determined by this object's mesh (if it is a mesh; otherwise, no BoundingAABB node will be generated)", default=False),
    "t3dAABBCustomSize__" : bpy.props.FloatVectorProperty(name="Size", description="Width (X), height (Y), and depth (Z) of the BoundingAABB node that will be created", min=0.0, default=[2,2,2]),
    "t3dTrianglesCustomBroadphaseEnabled__" : bpy.props.BoolProperty(name="Custom Broadphase Size", description="If enabled, you can manually set the BoundingTriangle's broadphase settings. If disabled, the BoundingTriangle's broadphase settings will be automatically determined by this object's size", default=False),
    "t3dTrianglesCustomBroadphaseGridSize__" : bpy.props.IntProperty(name="Broadphase Cell Size", description="How large the cells are in the broadphase collision grid (a cell size of 0 disables broadphase collision)", min=0, default=20),
    "t3dCapsuleCustomEnabled__" : bpy.props.BoolProperty(name="Custom Capsule Size", description="If enabled, you can manually set the BoundingCapsule node's size properties. If disabled, the Capsule's size will be automatically determined by this object's mesh (if it is a mesh; otherwise, no BoundingCapsule node will be generated)", default=False),
    "t3dCapsuleCustomRadius__" : bpy.props.FloatProperty(name="Radius", description="The radius of the BoundingCapsule node", min=0.0, default=0.5),
    "t3dCapsuleCustomHeight__" : bpy.props.FloatProperty(name="Height", description="The height of the BoundingCapsule node", min=0.0, default=2),
    "t3dSphereCustomEnabled__" : bpy.props.BoolProperty(name="Custom Sphere Size", description="If enabled, you can manually set the BoundingSphere node's radius. If disabled, the Sphere's size will be automatically determined by this object's mesh (if it is a mesh; otherwise, no BoundingSphere node will be generated)", default=False),
    "t3dSphereCustomRadius__" : bpy.props.FloatProperty(name="Radius", description="Radius of the BoundingSphere node that will be created", min=0.0, default=1),
    "t3dGameProperties__" : bpy.props.CollectionProperty(type=t3dGamePropertyItem__),
    "t3dObjectType__" : bpy.props.EnumProperty(items=objectTypes, name="Object Type", description="The type of object this is"),
    "t3dAutoBatch__" : bpy.props.EnumProperty(items=batchModes, name="Auto Batch", description="Whether objects should be automatically batched together; for dynamically batched objects, they can only have one, common Material. For statically merged objects, they can have however many materials"),
    "t3dAutoSubdivide__" : bpy.props.BoolProperty(name="Auto-Subdivide Faces", description="If enabled, Tetra3D will do its best to loop cut edges that are too large before export"),
    "t3dAutoSubdivideSize__" : bpy.props.FloatProperty(name="Max Edge Length", description="The maximum length an edge is allowed to be before automatically cutting prior to export", min=0.0, default=1.0),
    "t3dSectorType__" : bpy.props.EnumProperty(items=listSectorTypes,name="Sector Type", description="The type of sector capability this object has; only used if rendered with a camera with Sector Rendering on"),
    "t3dSectorTypeOverride__" : bpy.props.BoolProperty(name="Override Sector Type", description="If the collection object should override the sector type for ALL its objects"),
}

####

keymaps = []

def register():
    
    bpy.utils.register_class(OBJECT_PT_tetra3d)
    bpy.utils.register_class(ACTION_PT_tetra3d)
    bpy.utils.register_class(MESH_PT_tetra3d)
    bpy.utils.register_class(RENDER_PT_tetra3d)
    bpy.utils.register_class(CAMERA_PT_tetra3d)
    bpy.utils.register_class(MATERIAL_PT_tetra3d)
    bpy.utils.register_class(WORLD_PT_tetra3d)
    bpy.utils.register_class(SCENE_PT_tetra3d)
    
    bpy.utils.register_class(OBJECT_OT_tetra3dAddProp)
    bpy.utils.register_class(OBJECT_OT_tetra3dDeleteProp)
    bpy.utils.register_class(OBJECT_OT_tetra3dReorderProps)
    bpy.utils.register_class(OBJECT_OT_tetra3dCopyProps)
    bpy.utils.register_class(OBJECT_OT_tetra3dCopyOneProperty)
    bpy.utils.register_class(OBJECT_OT_tetra3dClearProps)
    bpy.utils.register_class(OBJECT_OT_tetra3dCopyNodePathToClipboard)
    bpy.utils.register_class(OBJECT_OT_tetra3dOverrideProp)

    bpy.utils.register_class(MATERIAL_OT_tetra3dMaterialCopyProps)
    bpy.utils.register_class(MATERIAL_OT_tetra3dAutoUV)

    bpy.utils.register_class(OBJECT_OT_tetra3dSetVector)
    
    bpy.utils.register_class(EXPORT_OT_tetra3d)
    bpy.utils.register_class(RENDER_OT_tetra3dQuickSetRenderResolution)

    bpy.utils.register_class(OBJECT_OT_tetra3dPlaySample)
    bpy.utils.register_class(OBJECT_OT_tetra3dStopSample)
    
    bpy.utils.register_class(t3dGamePropertyItem__)
    bpy.utils.register_class(OBJECT_OT_tetra3dSetAnimationInterpolation)
    bpy.utils.register_class(OBJECT_OT_tetra3dSetAnimationInterpolationAll)
    bpy.utils.register_class(OBJECT_OT_tetra3dFocusObject)
    bpy.utils.register_class(OBJECT_OT_tetra3dSearchStringProperties)

    for propName, prop in objectProps.items():
        setattr(bpy.types.Object, propName, prop)

    # We don't actually need to store or export the FOV; we just modify the camera's actual field of view (angle) property
    bpy.types.Camera.t3dFOV__ = bpy.props.IntProperty(name="FOV", description="Vertical field of view", default=75,
    get=getFOV, set=setFOV, min=1, max=179)
    
    bpy.types.Camera.t3dSectorRendering__ = bpy.props.BoolProperty(name="Sector-based Rendering", description="Whether scenes should be rendered according to sector or not", default=False)
    
    bpy.types.Camera.t3dSectorRenderDepth__ = bpy.props.IntProperty(name="Sector Render Depth", description="How many sector neighbors are rendered at a time", default=1, min=0)

    bpy.types.Scene.t3dSectorDetectionType__ = bpy.props.EnumProperty(items=sectorDetectionType, name="Sector Detection Type", description="How sector neighbors should be determined", default='VERTICES', 
    get=getSectorDetectionType, set=setSectorDetectionType)

    bpy.types.Scene.t3dGameProperties__ = objectProps["t3dGameProperties__"]

    bpy.types.Action.t3dGameProperties__ = objectProps["t3dGameProperties__"]
    bpy.types.Action.t3dRelativeMotion__ = bpy.props.BoolProperty(name="Relative Motion", description="Whether the animation's movements happen relative to the starting position and orientation of the object or not", default=False)

    perspectiveDescription = ("Whether the game should be rendered with perspective-corrected "
    "texture mapping or not. When enabled, it will look more like modern 3D texturing; when disabled, "
    "it will look like PS1 affine texture mapping (which is the default)."
    "This feature is experimental / not perfect currently (it looks fuzzy, and triangles aren't clipped properly at the edges of the viewport, "
    "which means you still get texture skewing when a triangle is largely offscreen)")
    bpy.types.Camera.t3dPerspectiveCorrectedTextureMapping__ = bpy.props.BoolProperty(name="Perspective-corrected Texture Mapping (Experimental)", description=perspectiveDescription, default=False)

    bpy.types.Camera.t3dMaxLightCount__ = bpy.props.IntProperty(name="Max light count", description="How many lights (sorted by distance to the camera, including ambient lights) should be used to light objects; if 0, then there is no limit", default=0, min = 0)

    bpy.types.Scene.t3dExportOnSave__ = bpy.props.BoolProperty(name="Export on Save", description="Whether the current file should export to GLTF on save or not", default=False, 
    get=getExportOnSave, set=setExportOnSave)
    
    bpy.types.Scene.t3dExportFilepath__ = bpy.props.StringProperty(name="Export Filepath", description="Filepath to export GLTF file. If left blank, it will export to the same directory as the blend file and will have the same filename; in this case, if the blend file has not been saved, nothing will happen", 
    default="", subtype="FILE_PATH", get=getExportFilepath, set=setExportFilepath)
    
    bpy.types.Scene.t3dExportFormat__ = bpy.props.EnumProperty(items=gltfExportTypes, name="Export Format", description="What format to export the file in", default="GLB",
    get=getExportFormat, set=setExportFormat)
    
    bpy.types.Scene.t3dExportCameras__ = bpy.props.BoolProperty(name="Export Cameras", description="Whether Blender should export cameras to the GLTF file", default=True,
    get=getExportCameras, set=setExportCameras)

    bpy.types.Scene.t3dExportLights__ = bpy.props.BoolProperty(name="Export Lights", description="Whether Blender should export lights to the GLTF file", default=True,
    get=getExportLights, set=setExportLights)

    bpy.types.Scene.t3dRenameInstancedObjects__ = bpy.props.BoolProperty(name="Rename Collection-Instanced Objects", description="Whether collection instances' names should be used for their instanced top-level objects", default=True,
    get=getRenameInstancedObjects, set=setRenameInstancedObjects)

    bpy.types.Scene.t3dPackTextures__ = bpy.props.BoolProperty(name="Pack Textures", description="Whether Blender should pack textures into the GLTF file on export", default=False,
    get=getPackTextures, set=setPackTextures)

    bpy.types.Scene.t3dRenderResolutionW__ = bpy.props.IntProperty(name="Render Width", description="How wide to render the game scene in pixels", default=640, min=0,
    get=getRenderResolutionW, set=setRenderResolutionW)

    bpy.types.Scene.t3dRenderResolutionH__ = bpy.props.IntProperty(name="Render Height", description="How tall to render the game scene in pixels", default=360, min=0,
    get=getRenderResolutionH, set=setRenderResolutionH)

    bpy.types.Scene.t3dPlaybackFPS__ = bpy.props.IntProperty(name="Playback FPS", description="Animation Playback Framerate (in Blender)", default=60, min=0,
    get=getPlaybackFPS, set=setPlaybackFPS)

    bpy.types.Scene.t3dAnimationSampling__ = bpy.props.BoolProperty(name="Sampled Animations", description="When enabled, animations are sampled (so you can use advanced techniques in your animations and then Blender will bake the results into your GLTF file). When disabled, only plain constant and linear animation keyframes (not cubic spline) will export. However, non-sampled animations export much more quickly than sampled animations, which means this option is useful when developing", default=True,
    get=getAnimationSampling, set=setAnimationSampling)

    bpy.types.Scene.t3dAnimationInterpolation__ = bpy.props.EnumProperty(
        items=[
            ("CONSTANT", "Constant", "Constant interpolation", 0, 0), 
            ("LINEAR", "Linear", "Linear interpolation", 0, 1), 
            ("BEZIER", "Bezier", "Bezier interpolation", 0, 2), 
            ("SINE", "Sine", "Sine interpolation", 0, 3), 
            ("QUAD", "Quad", "Quad interpolation", 0, 4), 
            ("CUBIC", "Cubic", "Cubic interpolation", 0, 5), 
            ("QUART", "Quart", "Quart interpolation", 0, 6), 
            ("QUINT", "Quint", "Quint interpolation", 0, 7), 
            ("EXPO", "Expo", "Exponential interpolation", 0, 8), 
            ("CIRC", "Circ", "Circ interpolation", 0, 7), 
            ("BACK", "Back", "Back interpolation", 0, 8), 
            ("BOUNCE", "Bounce", "Bounce interpolation", 0, 9), 
            ("ELASTIC", "Elastic", "Elastic interpolation", 0, 10),
        ], 
        name="Type", 
        description="What type to use for applying interpolation", 
        default="BEZIER",
        get=getAnimationInterpolation,
        set=setAnimationInterpolation)

    bpy.types.Scene.t3dExpandGameProps__ = bpy.props.BoolProperty(name="Expand Game Properties", default=True)
    bpy.types.Scene.t3dExpandOverrideProps__ = bpy.props.BoolProperty(name="Expand Overridden Properties", default=True)

    bpy.types.Material.t3dMaterialColor__ = bpy.props.FloatVectorProperty(name="Material Color", description="Material modulation color", default=[1,1,1,1], subtype="COLOR", size=4, step=1, min=0, max=1)
    bpy.types.Material.t3dMaterialShadeless__ = bpy.props.BoolProperty(name="Shadeless", description="Whether lighting should affect this material", default=False)
    bpy.types.Material.t3dMaterialFogless__ = bpy.props.BoolProperty(name="Fogless", description="Whether fog affects this material", default=False)
    bpy.types.Material.t3dBlendMode__ = bpy.props.EnumProperty(items=materialBlendModes, name="Blend Mode", description="Composite mode (i.e. additive, multiplicative, etc) for this material", default="DEFAULT")
    bpy.types.Material.t3dTransparencyMode__ = bpy.props.EnumProperty(items=materialTransparencyModes, name="Transparency Mode", description="Transparency mode for this material", default="AUTO")
    bpy.types.Material.t3dBillboardMode__ = bpy.props.EnumProperty(items=materialBillboardModes, name="Billboarding Mode", description="Billboard mode (i.e. if the object with this material should rotate to face the camera) for this material; doesn't take effect on armature skinned meshes", default="NONE")
    bpy.types.Material.t3dCustomDepthOn__ = bpy.props.BoolProperty(name="Custom Depth", description="Whether custom depth offsetting should be enabled", default=False)
    bpy.types.Material.t3dCustomDepthValue__ = bpy.props.FloatProperty(name="Depth Offset Value", description="How far in world units the material should offset when rendering (negative values are closer to the camera, positive values are further)")
    bpy.types.Material.t3dMaterialLightingMode__ = bpy.props.EnumProperty(items=materialLightingModes, name="Lighting mode", description="How materials should be lit", default="DEFAULT")
    bpy.types.Material.t3dVisible__ = bpy.props.BoolProperty(name="Visible", description="Whether this material is visible", default=True)

    bpy.types.Material.t3dAutoUV__ = bpy.props.BoolProperty(name="Auto UV-Map", description="If the UV map of the faces that use this material should automatically be Cube Projection UV mapped when exiting edit mode")
    bpy.types.Material.t3dAutoUVUnitSize__ = bpy.props.FloatProperty(name="Unit Size", description="How many Blender Units equates to one texture size", default=4.0, update=autoUVChange, step=5)
    bpy.types.Material.t3dAutoUVRotation__ = bpy.props.FloatProperty(name="Rotation", description="How many degrees to rotate the UVs after projection", step = 500, update=autoUVChange)
    bpy.types.Material.t3dAutoUVOffset__ = bpy.props.FloatVectorProperty(name="Offset", description="How many texels to offset the texture after projection", default=[0,0], size=2, update=autoUVChange, step=1)
        
    bpy.types.Material.t3dGameProperties__ = objectProps["t3dGameProperties__"]

    bpy.types.Mesh.t3dUniqueMesh__ = bpy.props.BoolProperty(name="Unique Mesh", description="Whether each Model that uses this mesh will have a unique clone of it or not. When enabled, any Models that use this mesh will clone the mesh on creation")
    bpy.types.Mesh.t3dUniqueMaterials__ = bpy.props.BoolProperty(name="Unique Materials", description="Whether each Model that uses this mesh's materials will have unique clones of them or not. When enabled, any Models that use this mesh will clone the materials on creation")
    
    bpy.types.World.t3dClearColor__ = bpy.props.FloatVectorProperty(name="Clear Color", description="Screen clear color; note that this won't actually be the background color automatically, but rather is simply set on the Scene.ClearColor property for you to use as you wish", default=[0.007, 0.008, 0.01, 1], subtype="COLOR", size=4, step=1, min=0, max=1)
    bpy.types.World.t3dSyncClearColor__ = bpy.props.BoolProperty(name="Sync Clear Color to World Color", description="If the clear color should be a copy of the world's color")
    bpy.types.World.t3dFogColor__ = bpy.props.FloatVectorProperty(name="Fog Color", description="The color of fog for this world", default=[0, 0, 0, 1], subtype="COLOR", size=4, step=1, min=0, max=1)
    bpy.types.World.t3dSyncFogColor__ = bpy.props.BoolProperty(name="Sync Fog Color to Clear Color", description="If the fog color should be a copy of the screen clear color")
    bpy.types.World.t3dFogMode__ = bpy.props.EnumProperty(items=worldFogCompositeModes, name="Fog Mode", description="Fog mode", default="OFF")
    bpy.types.World.t3dFogDithered__ = bpy.props.FloatProperty(name="Fog Dither Size", description="How large bayer matrix dithering is when using fog. If set to 0, dithering is disabled", default=0, min=0, step=1)

    bpy.types.World.t3dFogCurve__ = bpy.props.EnumProperty(items=worldFogCurveTypes, name="Fog Curve", description="What curve to use for the fog's gradience", default="LINEAR")
    bpy.types.World.t3dFogRangeStart__ = bpy.props.FloatProperty(name="Fog Range Start", description="With 0 being the near plane and 1 being the far plane of the camera, how far in should the fog start to appear", min=0.0, max=1.0, default=0, get=fogRangeStartGet, set=fogRangeStartSet)
    bpy.types.World.t3dFogRangeEnd__ = bpy.props.FloatProperty(name="Fog Range End", description="With 0 being the near plane and 1 being the far plane of the camera, how far out should the fog be at maximum opacity", min=0.0, max=1.0, default=1, get=fogRangeEndGet, set=fogRangeEndSet)

    # Handlers and callbacks

    if not exportOnSave in bpy.app.handlers.save_post:
        bpy.app.handlers.save_post.append(exportOnSave)
    if not onLoad in bpy.app.handlers.load_post:
        bpy.app.handlers.load_post.append(onLoad)

    keyconfig = bpy.context.window_manager.keyconfigs.addon

    kc = keyconfig.keymaps.new(name="Window", space_type='EMPTY')
    shortcut = kc.keymap_items.new("export.tetra3dgltf", 'E', 'PRESS', ctrl=True, shift=True)

    global keymaps
    keymaps.append((kc, shortcut))

def unregister():
    bpy.utils.unregister_class(OBJECT_PT_tetra3d)
    bpy.utils.unregister_class(ACTION_PT_tetra3d)
    bpy.utils.unregister_class(MESH_PT_tetra3d)
    bpy.utils.unregister_class(RENDER_PT_tetra3d)
    bpy.utils.unregister_class(CAMERA_PT_tetra3d)
    bpy.utils.unregister_class(MATERIAL_PT_tetra3d)
    bpy.utils.unregister_class(WORLD_PT_tetra3d)
    bpy.utils.unregister_class(SCENE_PT_tetra3d)

    bpy.utils.unregister_class(OBJECT_OT_tetra3dAddProp)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dDeleteProp)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dReorderProps)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dCopyProps)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dCopyOneProperty)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dClearProps)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dCopyNodePathToClipboard)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dOverrideProp)

    bpy.utils.unregister_class(OBJECT_OT_tetra3dSetVector)

    bpy.utils.unregister_class(MATERIAL_OT_tetra3dMaterialCopyProps)
    bpy.utils.unregister_class(MATERIAL_OT_tetra3dAutoUV)

    bpy.utils.unregister_class(EXPORT_OT_tetra3d)
    
    bpy.utils.unregister_class(RENDER_OT_tetra3dQuickSetRenderResolution)

    bpy.utils.unregister_class(t3dGamePropertyItem__)

    if currentlyPlayingAudioHandle:
        currentlyPlayingAudioHandle.stop()

    bpy.utils.unregister_class(OBJECT_OT_tetra3dPlaySample)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dStopSample)

    bpy.utils.unregister_class(OBJECT_OT_tetra3dSetAnimationInterpolationAll)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dSetAnimationInterpolation)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dFocusObject)

    bpy.utils.unregister_class(OBJECT_OT_tetra3dSearchStringProperties)
    
    for propName in objectProps.keys():
        delattr(bpy.types.Object, propName)

    del bpy.types.Scene.t3dGameProperties__
    del bpy.types.Action.t3dGameProperties__
    del bpy.types.Action.t3dRelativeMotion__

    del bpy.types.Scene.t3dSectorDetectionType__
    del bpy.types.Scene.t3dExportOnSave__
    del bpy.types.Scene.t3dExportFilepath__
    del bpy.types.Scene.t3dExportFormat__
    del bpy.types.Scene.t3dExportCameras__
    del bpy.types.Scene.t3dExportLights__
    del bpy.types.Scene.t3dRenameInstancedObjects__
    del bpy.types.Scene.t3dPackTextures__
    del bpy.types.Scene.t3dAnimationSampling__
    del bpy.types.Scene.t3dAnimationInterpolation__

    del bpy.types.Scene.t3dRenderResolutionW__
    del bpy.types.Scene.t3dRenderResolutionH__
    del bpy.types.Scene.t3dPlaybackFPS__

    del bpy.types.Scene.t3dExpandGameProps__
    del bpy.types.Scene.t3dExpandOverrideProps__

    del bpy.types.Material.t3dMaterialColor__
    del bpy.types.Material.t3dMaterialShadeless__
    del bpy.types.Material.t3dMaterialFogless__
    del bpy.types.Material.t3dBlendMode__
    del bpy.types.Material.t3dBillboardMode__
    del bpy.types.Material.t3dTransparencyMode__
    del bpy.types.Material.t3dMaterialLightingMode__

    del bpy.types.Material.t3dCustomDepthOn__
    del bpy.types.Material.t3dCustomDepthValue__
    del bpy.types.Material.t3dGameProperties__
    del bpy.types.Material.t3dAutoUV__
    del bpy.types.Material.t3dAutoUVUnitSize__
    del bpy.types.Material.t3dAutoUVRotation__
    del bpy.types.Material.t3dVisible__

    del bpy.types.World.t3dClearColor__
    del bpy.types.World.t3dFogColor__
    del bpy.types.World.t3dFogMode__
    del bpy.types.World.t3dFogRangeStart__
    del bpy.types.World.t3dFogRangeEnd__
    del bpy.types.World.t3dFogDithered__
    del bpy.types.World.t3dFogCurve__

    del bpy.types.Mesh.t3dUniqueMesh__
    del bpy.types.Mesh.t3dUniqueMaterials__

    del bpy.types.Camera.t3dSectorRendering__
    del bpy.types.Camera.t3dSectorRenderDepth__
    del bpy.types.Camera.t3dPerspectiveCorrectedTextureMapping__
    del bpy.types.Camera.t3dFOV__
    del bpy.types.Camera.t3dMaxLightCount__

    if exportOnSave in bpy.app.handlers.save_post:
        bpy.app.handlers.save_post.remove(exportOnSave)
    if onLoad in bpy.app.handlers.load_post:
        bpy.app.handlers.load_post.remove(onLoad)

    global keymaps

    for keymap, shortcut in keymaps:
        keymap.keymap_items.remove(shortcut)

    keymaps.clear()

    bpy.msgbus.clear_by_owner("tetra3d")

if __name__ == "__main__":
    register()
