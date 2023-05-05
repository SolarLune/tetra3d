# Add-on for Tetra3D > Blender exporting

import bpy, os, bmesh, math, aud
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
    "version" : (0, 11, 2),
    "support" : "COMMUNITY",
    "doc_url" : "https://github.com/SolarLune/Tetra3d/wiki/Blender-Addon",
}

objectTypes = [
    ("MESH", "Mesh", "A standard, visible mesh object", 0, 0),
    ("GRID", "Grid", "A grid object; not visualized or 'physically present'. The vertices in Blender become grid points in Tetra3D; the edges become their connections", 0, 1),
]

boundsTypes = [
    ("NONE", "No Bounds", "No collision will be created for this object.", 0, 0),
    ("AABB", "AABB", "An AABB (axis-aligned bounding box). If the size isn't customized, it will be big enough to fully contain the mesh of the current object. Currently buggy when resolving intersections between AABB or other Triangle Nodes", 0, 1),
    ("CAPSULE", "Capsule", "A capsule, which can rotate. If the radius and height are not set, it will have a radius and height to fully contain the current object", 0, 2),
    ("SPHERE", "Sphere", "A sphere. If the radius is not custom set, it will have a large enough radius to fully contain the provided object", 0, 3),
    ("TRIANGLES", "Triangle Mesh", "A triangle mesh bounds type. Only works on mesh-type objects (i.e. an Empty won't generate a BoundingTriangles). Accurate, but slow. Currently buggy when resolving intersections between AABB or other Triangle Nodes", 0, 4),
]

gltfExportTypes = [
    ("GLB", ".glb", "Exports a single file, with all data packed in binary form. Most efficient and portable, but more difficult to edit later", 0, 0),
    ("GLTF_SEPARATE", ".gltf + .bin + textures", "Exports multiple files, with separate JSON, binary and texture data. Easiest to edit later - Note that Tetra3D doesn't support this properly currently", 0, 1),
    ("GLTF_EMBEDDED", ".gltf", "Exports a single file, with all data packed in JSON. Less efficient than binary, but easier to edit later", 0, 2),
]

materialCompositeModes = [
    ("DEFAULT", "Default", "Blends the destination by the material's color modulated by the material's alpha value. The default alpha-blending composite mode. Also known as CompositeModeSourceOver", 0, 0),
    ("ADDITIVE", "Additive", "Adds the material's color to the destination. Also known as CompositeModeLighter", 0, 1),
    # ("MULTIPLY", "Multiply", "Multiplies the material's color by the destination. Also known as CompositeModeMultiply", 0, 2),
    ("CLEAR", "Clear", "Anywhere the material draws is cleared instead; useful to 'punch through' a scene to show the blank alpha zero. Also known as CompositeModeClear", 0, 3),
]

materialBillboardModes = [
    ("NONE", "None", "No billboarding - the (unskinned) object with this material does not rotate to face the camera.", 0, 0),
    ("XZ", "X/Z", "X/Z billboarding - the (unskinned) object with this material rotates to the face the camera only on the X and Z axes (not the Y axis).", 0, 1),
    ("FULL", "Full", "Full billboarding - the (unskinned) object rotates fully to face the camera.", 0, 2),
]

worldFogCompositeModes = [
    ("OFF", "Off", "No fog. Object colors aren't changed with distance from the camera", 0, 0),
    ("ADDITIVE", "Additive", "Additive fog - this fog mode brightens objects in the distance, with full effect being adding the color given to the object's color at maximum distance (according to the camera's far range)", 0, 1),
    ("SUBTRACT", "Subtractive", "Subtractive fog - this fog mode darkens objects in the distance, with full effect being subtracting the object's color by the fog color at maximum distance (according to the camera's far range)", 0, 2),
    ("OVERWRITE", "Overwrite", "Overwrite fog - this fog mode overwrites the object's color with the fog color, with maximum distance being the camera's far distance", 0, 3),
    ("TRANSPARENT", "Transparent", "Transparent fog - this fog mode fades the object out over distance, such that at maximum distance / fog range, the object is wholly transparent.", 0, 4),
]

worldFogCurveTypes = [
    ("LINEAR", "Smooth", "Smooth fog (Ease: Linear); this goes from 0% in the near range to 100% in the far range evenly", "LINCURVE", 0),
    ("OUTCIRC", "Dense", "Dense fog (Ease: Out Circ); fog will increase aggressively in the near range, ramping up to 100% at the far range", "SPHERECURVE", 1),
    ("INCIRC", "Light", "Light fog (Ease: In Circ); fog will increase aggressively towards the far range, ramping up to 100% at the far range", "SHARPCURVE", 2),
]

gamePropTypes = [
    ("bool", "Bool", "Boolean data type", 0, 0),
    ("int", "Int", "Int data type", 0, 1),
    ("float", "Float", "Float data type", 0, 2),
    ("string", "String", "String data type", 0, 3),
    ("reference", "Object", "Object reference data type; converted to a string composed as follows on export - [SCENE NAME]:[OBJECT NAME]", 0, 4),
    ("color", "Color", "Color data type", 0, 5),
    ("vector3d", "3D Vector", "3D vector data type", 0, 6),
    ("file", "Filepath", "Filepath", 0, 7),
]

batchModes = [ 
    ("OFF", "Off", "No automatic batching", 0, 0), 
    ("DYNAMIC", "Dynamic Batching", "Dynamic batching based off of one material (the first one)", 0, 1), 
    ("STATIC", "Static Merging", "Static merging; merged objects cannot move or deviate in any way. After automatic static merging, the merged models will be automatically set to invisible", 0, 2),
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

        target = getattr(context, self.mode)
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

        if self.buttonMode == "object location":
            target.t3dGameProperties__[self.index].valueVector3D = context.object.location
        elif self.buttonMode == "3D cursor":
            target.t3dGameProperties__[self.index].valueVector3D = context.scene.cursor.location

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


class OBJECT_OT_tetra3dOverrideProp(bpy.types.Operator):
    bl_idname = "object.tetra3doverrideprop"
    bl_label = "Apply Game Property"
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
    bl_options = {'REGISTER'}

    filepath : bpy.props.StringProperty()

    def execute(self, context):
        
        global currentlyPlayingAudioHandle, audioPaused

        if currentlyPlayingAudioHandle:
            currentlyPlayingAudioHandle.pause()
            audioPaused = True

        return {'FINISHED'}

def objectNodePath(object):

    p = object.name

    if object.parent:
        p = objectNodePath(object.parent) + "/" + object.name

    return p

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
            
            row = box.row()
            row.enabled = context.object.t3dObjectType__ == 'MESH'
            row.prop(context.object, "t3dSector__")

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

        if context.object.instance_type == "COLLECTION" and context.object.instance_collection is not None:

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

                        for propIndex, prop in enumerate(object.t3dGameProperties__):
                            
                            row = box.row()
                            row.label(text=prop.name)

                            op = row.operator(OBJECT_OT_tetra3dOverrideProp.bl_idname)
                            op.objectIndex = objectIndex
                            op.propIndex = propIndex

                            row = box.row()
                            row.enabled = False
                            row.prop(prop, "name")
                            handleT3DProperty(propIndex, box, prop, "object", False)

        row = self.layout.row()
        row.label(text="Game Properties")
        row.prop(context.scene, "t3dExpandGameProps__", icon="TRIA_DOWN" if context.scene.t3dExpandGameProps__ else "TRIA_RIGHT", icon_only=True, emboss=False)


        if context.scene.t3dExpandGameProps__:

            row = self.layout.row()

            add = row.operator("object.tetra3daddprop", text="Add Game Property", icon="PLUS")
            add.mode = "object"

            row.operator("object.tetra3dcopyprops", text="Overwrite All Game Properties", icon="COPYDOWN")

            for index, prop in enumerate(context.object.t3dGameProperties__):
                box = self.layout.box()
                row = box.row()
                row.prop(prop, "name")
                
                moveUpOptions = row.operator(OBJECT_OT_tetra3dReorderProps.bl_idname, text="", icon="TRIA_UP")
                moveUpOptions.index = index
                moveUpOptions.moveUp = True
                moveUpOptions.mode = "object"

                moveDownOptions = row.operator(OBJECT_OT_tetra3dReorderProps.bl_idname, text="", icon="TRIA_DOWN")
                moveDownOptions.index = index
                moveDownOptions.moveUp = False
                moveDownOptions.mode = "object"

                copy = row.operator(OBJECT_OT_tetra3dCopyOneProperty.bl_idname, text="", icon="COPYDOWN")
                copy.index = index

                deleteOptions = row.operator(OBJECT_OT_tetra3dDeleteProp.bl_idname, text="", icon="TRASH")
                deleteOptions.index = index
                deleteOptions.mode = "object"

                handleT3DProperty(index, box, prop, "object")

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

        for index, prop in enumerate(context.scene.t3dGameProperties__):
            box = self.layout.box()
            row = box.row()
            row.prop(prop, "name")
            
            moveUpOptions = row.operator(OBJECT_OT_tetra3dReorderProps.bl_idname, text="", icon="TRIA_UP")
            moveUpOptions.index = index
            moveUpOptions.moveUp = True
            moveUpOptions.mode = "scene"

            moveDownOptions = row.operator(OBJECT_OT_tetra3dReorderProps.bl_idname, text="", icon="TRIA_DOWN")
            moveDownOptions.index = index
            moveDownOptions.moveUp = False
            moveDownOptions.mode = "scene"

            deleteOptions = row.operator(OBJECT_OT_tetra3dDeleteProp.bl_idname, text="", icon="TRASH")
            deleteOptions.index = index
            deleteOptions.mode = "scene"

            handleT3DProperty(index, box, prop, "scene")
        
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


def handleT3DProperty(index, box, prop, operatorType, enabled=True):

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
    elif prop.valueType == "reference":
        row.prop(prop, "valueReferenceScene")
        if prop.valueReferenceScene != None:
            row.prop_search(prop, "valueReference", prop.valueReferenceScene, "objects")
        else:
            row.prop(prop, "valueReference")
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
        row = self.layout.row()
        row.prop(context.material, "blend_method")
        row = self.layout.row()
        row.prop(context.material, "t3dCompositeMode__")
        row = self.layout.row()
        row.prop(context.material, "t3dBillboardMode__")

        if context.object.active_material != None:

            row = self.layout.row()
            add = row.operator("object.tetra3daddprop", text="Add Game Property", icon="PLUS")
            add.mode = "material"

            row.operator("material.tetra3dcopyprops", icon="COPYDOWN")
            
            for index, prop in enumerate(context.object.active_material.t3dGameProperties__):
                box = self.layout.box()
                row = box.row()
                row.prop(prop, "name")
                
                moveUpOptions = row.operator(OBJECT_OT_tetra3dReorderProps.bl_idname, text="", icon="TRIA_UP")
                moveUpOptions.index = index
                moveUpOptions.moveUp = True
                moveUpOptions.mode = "material"

                moveDownOptions = row.operator(OBJECT_OT_tetra3dReorderProps.bl_idname, text="", icon="TRIA_DOWN")
                moveDownOptions.index = index
                moveDownOptions.moveUp = False
                moveDownOptions.mode = "material"

                deleteOptions = row.operator(OBJECT_OT_tetra3dDeleteProp.bl_idname, text="", icon="TRASH")
                deleteOptions.index = index
                deleteOptions.mode = "material"

                handleT3DProperty(index, box, prop, "material")

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
        row.prop(context.world, "t3dClearColor__")
        box = self.layout.box()
        row = box.row()
        row.prop(context.world, "t3dFogMode__")

        if context.world.t3dFogMode__ != "OFF":

            box.prop(context.world, "t3dFogCurve__")

            if context.world.t3dFogMode__ != "TRANSPARENT" and not context.world.t3dSyncFogColor__:
                box.prop(context.world, "t3dFogColor__")

            box.prop(context.world, "t3dSyncFogColor__")

            box.prop(context.world, "t3dFogDithered__")            
            box.prop(context.world, "t3dFogRangeStart__", slider=True)
            box.prop(context.world, "t3dFogRangeEnd__", slider=True)
        
# The idea behind "globalget and set" is that we're setting properties on the first scene (which must exist), and getting any property just returns the first one from that scene
def globalGet(propName):
    if propName in bpy.data.scenes[0]:
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
        box.prop(context.scene, "t3dPackTextures__")
        box.prop(context.scene, "t3dExportCameras__")
        box.prop(context.scene, "t3dExportLights__")

        box = self.layout.box()
        box.prop(context.scene, "t3dSectorRendering__")
        row = box.row()
        row.enabled = context.scene.t3dSectorRendering__
        row.prop(context.scene, "t3dSectorRenderDepth__")

        row = self.layout.row()
        row.prop(context.scene, "t3dRenderResolutionW__")
        row.prop(context.scene, "t3dRenderResolutionH__")
        row = self.layout.row()
        row.label(text="Animation Playback Framerate (in Blender):")
        row = self.layout.row()
        row.prop(context.scene, "t3dPlaybackFPS__")

def export():
    scene = bpy.context.scene

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
    
    if scene.t3dExportFormat__ == "GLB":
        ending = ".glb"
    elif scene.t3dExportFormat__ == "GLTF_SEPARATE" or scene.t3dExportFormat__ == "GLTF_EMBEDDED":
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

        if "t3dClearColor__" in world:
            worldData["clear color"] = world.t3dClearColor__
        if "t3dFogMode__" in world:
            worldData["fog mode"] = world.t3dFogMode__
        if "t3dFogDithered__" in world:
            worldData["dithered transparency"] = world.t3dFogDithered__
        if "t3dFogCurve__" in world:
            worldData["fog curve"] = world.t3dFogCurve__
        if "t3dFogColor__" in world:
            if "t3dSyncFogColor__" in world and world["t3dSyncFogColor__"] and "t3dClearColor__" in world:
                worldData["fog color"] = world.t3dClearColor__
            else:
                worldData["fog color"] = world.t3dFogColor__
        if "t3dFogRangeStart__" in world:
            worldData["fog range start"] = world.t3dFogRangeStart__
        if "t3dFogRangeEnd__" in world:
            worldData["fog range end"] = world.t3dFogRangeEnd__

        worlds[world.name] = worldData

    globalSet("t3dWorlds__", worlds)

    currentFrame = {}

    autoSubdivides = {}

    for scene in bpy.data.scenes:

        currentFrame[scene] = scene.frame_current

        if scene.users > 0:

            if scene.world:
                scene["t3dCurrentWorld__"] = scene.world.name

            for layer in scene.view_layers:
                for obj in layer.objects:

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
                        
                        if len(obj.data.color_attributes) > 0:
                            vertexColors = [layer.name for layer in obj.data.color_attributes]
                            obj.data["t3dVertexColorNames__"] = vertexColors
                            obj.data["t3dActiveVertexColorIndex__"] = obj.data.color_attributes.render_color_index

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

                    if obj.instance_type == "COLLECTION":
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
                            if workingEdge is None:
                                workingEdge = edge

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
                "time": marker.frame / globalGet("t3dPlaybackFPS__"),
            }
            markers.append(markerInfo)
        if len(markers) > 0:
            action["t3dMarkers__"] = markers

    # We force on exporting of Extra values because otherwise, values from Blender would not be able to be exported.
    # export_apply=True to ensure modifiers are applied.
    bpy.ops.export_scene.gltf(
        filepath=newPath, 
        # use_active_scene=True, # Blender's GLTF exporter's kinda thrashed when it comes to multiple scenes, so it might be better to export each scene as its own GLTF file...?
        export_format=scene.t3dExportFormat__, 
        export_cameras=scene.t3dExportCameras__, 
        export_lights=scene.t3dExportLights__, 
        export_keep_originals=not scene.t3dPackTextures__,
        export_attributes=True,
        
        export_current_frame=False,
        export_nla_strips=True,
        export_animations=True,
        export_frame_range = False,

        export_extras=True,
        export_yup=True,
        export_apply=True,
        convert_lighting_mode="COMPAT", # We want to use the compatible lighting model, not the "realistic" / real-world-accurate one
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
                        if obj.t3dObjectType__ == 'GRID':
                            del(obj.data["t3dGrid__"])
                            del(obj["t3dGridConnections__"])
                            del(obj["t3dGridEntries__"])
                            obj.data = ogGrids[obj] # Restore the mesh reference afterward


    for action in bpy.data.actions:
        if "t3dMarkers__" in action:
            del(action["t3dMarkers__"])

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

    return True

@persistent
def exportOnSave(dummy):
    
    if globalGet("t3dExportOnSave__"):
        export()

@persistent
def onLoad(dummy):

    global currentlyPlayingAudioHandle, currentlyPlayingAudioName, audioPaused

    if currentlyPlayingAudioHandle:
        currentlyPlayingAudioHandle.stop()
        currentlyPlayingAudioHandle = None
        currentlyPlayingAudioName = ""
        audioPaused = False


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
    "t3dSector__" : bpy.props.BoolProperty(name="Is a Sector", description="If enabled, this identifies a mesh as being a Sector. When this is the case and sector-based rendering has been enabled on a Camera, this allows you to granularly restrict how far the camera renders, not based off of near/far plane only, but also based off of which sector you're in and how many sectors in you can look")
}


def getSectorRendering(self):
    s = globalGet("t3dSectorRendering__")
    if s is None:
        s = False
    return s

def setSectorRendering(self, value):
    globalSet("t3dSectorRendering__", value)


def getSectorRenderDepth(self):
    s = globalGet("t3dSectorRenderDepth__")
    if s is None:
        s = False
    return s

def setSectorRenderDepth(self, value):
    globalSet("t3dSectorRenderDepth__", value)

#####

def getRenderResolutionW(self):
    s = globalGet("t3dRenderResolutionW__")
    if s is None:
        s = 640
    bpy.context.scene.render.resolution_x = s
    return s

def setRenderResolutionW(self, value):
    globalSet("t3dRenderResolutionW__", value)
    bpy.context.scene.render.resolution_x = value

#####

def getRenderResolutionH(self):
    s = globalGet("t3dRenderResolutionH__")
    if s is None:
        s = 360
    bpy.context.scene.render.resolution_y = s
    return s

def setRenderResolutionH(self, value):
    globalSet("t3dRenderResolutionH__", value)
    bpy.context.scene.render.resolution_y = value

######

def getPlaybackFPS(self):
    s = globalGet("t3dPlaybackFPS__")
    if s is None:
        s = 60
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
    s = globalGet("t3dExportOnSave__")
    if s is None:
        s = False
    return s

def setExportOnSave(self, value):
    globalSet("t3dExportOnSave__", value)



def getExportFilepath(self):
    fp = globalGet("t3dExportFilepath__")
    if fp is None:
        fp = ""
    return fp

def setExportFilepath(self, value):
    globalSet("t3dExportFilepath__", value)



def getExportFormat(self):
    f = globalGet("t3dExportFormat__")
    if f is None:
        f = 0
    return f

def setExportFormat(self, value):
    globalSet("t3dExportFormat__", value)



def getExportCameras(self):
    c = globalGet("t3dExportCameras__")
    if c is None:
        c = True
    return c

def setExportCameras(self, value):
    globalSet("t3dExportCameras__", value)



def getExportLights(self):
    l = globalGet("t3dExportLights__")
    if l is None:
        l = True
    return l

def setExportLights(self, value):
    globalSet("t3dExportLights__", value)


def getPackTextures(self):
    l = globalGet("t3dPackTextures__")
    if l is None:
        l = False
    return l

def setPackTextures(self, value):
    globalSet("t3dPackTextures__", value)


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

def register():
    
    bpy.utils.register_class(OBJECT_PT_tetra3d)
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

    bpy.utils.register_class(OBJECT_OT_tetra3dSetVector)
    
    bpy.utils.register_class(EXPORT_OT_tetra3d)

    bpy.utils.register_class(OBJECT_OT_tetra3dPlaySample)
    bpy.utils.register_class(OBJECT_OT_tetra3dStopSample)
    
    bpy.utils.register_class(t3dGamePropertyItem__)

    for propName, prop in objectProps.items():
        setattr(bpy.types.Object, propName, prop)

    # We don't actually need to store or export the FOV; we just modify the camera's actual field of view (angle) property
    bpy.types.Camera.t3dFOV__ = bpy.props.IntProperty(name="FOV", description="Vertical field of view", default=75,
    get=getFOV, set=setFOV, min=1, max=179)
    
    bpy.types.Scene.t3dGameProperties__ = objectProps["t3dGameProperties__"]

    bpy.types.Scene.t3dSectorRendering__ = bpy.props.BoolProperty(name="Sector-based Rendering", description="Whether scenes should be rendered according to sector or not", default=False, 
    get=getSectorRendering, set=setSectorRendering)

    bpy.types.Scene.t3dSectorRenderDepth__ = bpy.props.IntProperty(name="Sector Render Depth", description="How many sector neighbors are rendered at a time", default=1, min=0,
    get=getSectorRenderDepth, set=setSectorRenderDepth)

    bpy.types.Scene.t3dExportOnSave__ = bpy.props.BoolProperty(name="Export on Save", description="Whether the current file should export to GLTF on save or not", default=False, 
    get=getExportOnSave, set=setExportOnSave)
    
    bpy.types.Scene.t3dExportFilepath__ = bpy.props.StringProperty(name="Export Filepath", description="Filepath to export GLTF file. If left blank, it will export to the same directory as the blend file and will have the same filename; in this case, if the blend file has not been saved, nothing will happen", 
    default="", subtype="FILE_PATH", get=getExportFilepath, set=setExportFilepath)
    
    bpy.types.Scene.t3dExportFormat__ = bpy.props.EnumProperty(items=gltfExportTypes, name="Export Format", description="What format to export the file in", default="GLTF_EMBEDDED",
    get=getExportFormat, set=setExportFormat)
    
    bpy.types.Scene.t3dExportCameras__ = bpy.props.BoolProperty(name="Export Cameras", description="Whether Blender should export cameras to the GLTF file", default=True,
    get=getExportCameras, set=setExportCameras)

    bpy.types.Scene.t3dExportLights__ = bpy.props.BoolProperty(name="Export Lights", description="Whether Blender should export lights to the GLTF file", default=True,
    get=getExportLights, set=setExportLights)

    bpy.types.Scene.t3dPackTextures__ = bpy.props.BoolProperty(name="Pack Textures", description="Whether Blender should pack textures into the GLTF file on export", default=False,
    get=getPackTextures, set=setPackTextures)

    bpy.types.Scene.t3dRenderResolutionW__ = bpy.props.IntProperty(name="Render Resolution Width", description="How wide to render the game scene", default=640, min=0,
    get=getRenderResolutionW, set=setRenderResolutionW)

    bpy.types.Scene.t3dRenderResolutionH__ = bpy.props.IntProperty(name="Render Resolution Height", description="How wide to render the game scene", default=360, min=0,
    get=getRenderResolutionH, set=setRenderResolutionH)

    bpy.types.Scene.t3dPlaybackFPS__ = bpy.props.IntProperty(name="Playback FPS", description="Animation Playback Framerate (in Blender)", default=60, min=0,
    get=getPlaybackFPS, set=setPlaybackFPS)

    bpy.types.Scene.t3dExpandGameProps__ = bpy.props.BoolProperty(name="Expand Game Properties", default=True)
    bpy.types.Scene.t3dExpandOverrideProps__ = bpy.props.BoolProperty(name="Expand Overridden Properties", default=True)

    bpy.types.Material.t3dMaterialColor__ = bpy.props.FloatVectorProperty(name="Material Color", description="Material modulation color", default=[1,1,1,1], subtype="COLOR", size=4, step=1, min=0, max=1)
    bpy.types.Material.t3dMaterialShadeless__ = bpy.props.BoolProperty(name="Shadeless", description="Whether lighting should affect this material", default=False)
    bpy.types.Material.t3dMaterialFogless__ = bpy.props.BoolProperty(name="Fogless", description="Whether fog affects this material", default=False)
    bpy.types.Material.t3dCompositeMode__ = bpy.props.EnumProperty(items=materialCompositeModes, name="Composite Mode", description="Composite mode (i.e. additive, multiplicative, etc) for this material", default="DEFAULT")
    bpy.types.Material.t3dBillboardMode__ = bpy.props.EnumProperty(items=materialBillboardModes, name="Billboarding Mode", description="Billboard mode (i.e. if the object with this material should rotate to face the camera) for this material", default="NONE")
    
    bpy.types.Material.t3dGameProperties__ = objectProps["t3dGameProperties__"]
    
    bpy.types.World.t3dClearColor__ = bpy.props.FloatVectorProperty(name="Clear Color", description="Screen clear color; note that this won't actually be the background color automatically, but rather is simply set on the Scene.ClearColor property for you to use as you wish", default=[0.007, 0.008, 0.01, 1], subtype="COLOR", size=4, step=1, min=0, max=1)
    bpy.types.World.t3dFogColor__ = bpy.props.FloatVectorProperty(name="Fog Color", description="The color of fog for this world", default=[0, 0, 0, 1], subtype="COLOR", size=4, step=1, min=0, max=1)
    bpy.types.World.t3dSyncFogColor__ = bpy.props.BoolProperty(name="Sync Fog Color to Clear Color", description="If the fog color should be a copy of the screen clear color")
    bpy.types.World.t3dFogMode__ = bpy.props.EnumProperty(items=worldFogCompositeModes, name="Fog Mode", description="Fog mode", default="OFF")
    bpy.types.World.t3dFogDithered__ = bpy.props.FloatProperty(name="Fog Dither Size", description="How large bayer matrix dithering is when using fog. If set to 0, dithering is disabled", default=0, min=0, step=1)

    bpy.types.World.t3dFogCurve__ = bpy.props.EnumProperty(items=worldFogCurveTypes, name="Fog Curve", description="What curve to use for the fog's gradience", default="LINEAR")
    bpy.types.World.t3dFogRangeStart__ = bpy.props.FloatProperty(name="Fog Range Start", description="With 0 being the near plane and 1 being the far plane of the camera, how far in should the fog start to appear", min=0.0, max=1.0, default=0, get=fogRangeStartGet, set=fogRangeStartSet)
    bpy.types.World.t3dFogRangeEnd__ = bpy.props.FloatProperty(name="Fog Range End", description="With 0 being the near plane and 1 being the far plane of the camera, how far out should the fog be at maximum opacity", min=0.0, max=1.0, default=1, get=fogRangeEndGet, set=fogRangeEndSet)

    if not exportOnSave in bpy.app.handlers.save_post:
        bpy.app.handlers.save_post.append(exportOnSave)
    if not onLoad in bpy.app.handlers.load_post:
        bpy.app.handlers.load_post.append(onLoad)
    
def unregister():
    bpy.utils.unregister_class(OBJECT_PT_tetra3d)
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

    bpy.utils.unregister_class(EXPORT_OT_tetra3d)
    
    bpy.utils.unregister_class(t3dGamePropertyItem__)

    if currentlyPlayingAudioHandle:
        currentlyPlayingAudioHandle.stop()

    bpy.utils.unregister_class(OBJECT_OT_tetra3dPlaySample)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dStopSample)
    
    for propName in objectProps.keys():
        delattr(bpy.types.Object, propName)

    del bpy.types.Scene.t3dGameProperties__

    del bpy.types.Scene.t3dSectorRendering__
    del bpy.types.Scene.t3dSectorRenderDepth__
    del bpy.types.Scene.t3dExportOnSave__
    del bpy.types.Scene.t3dExportFilepath__
    del bpy.types.Scene.t3dExportFormat__
    del bpy.types.Scene.t3dExportCameras__
    del bpy.types.Scene.t3dExportLights__
    del bpy.types.Scene.t3dPackTextures__

    del bpy.types.Scene.t3dRenderResolutionW__
    del bpy.types.Scene.t3dRenderResolutionH__
    del bpy.types.Scene.t3dPlaybackFPS__

    del bpy.types.Scene.t3dExpandGameProps__
    del bpy.types.Scene.t3dExpandOverrideProps__

    del bpy.types.Material.t3dMaterialColor__
    del bpy.types.Material.t3dMaterialShadeless__
    del bpy.types.Material.t3dMaterialFogless__
    del bpy.types.Material.t3dCompositeMode__
    del bpy.types.Material.t3dBillboardMode__
    del bpy.types.Material.t3dGameProperties__

    del bpy.types.World.t3dClearColor__
    del bpy.types.World.t3dFogColor__
    del bpy.types.World.t3dFogMode__
    del bpy.types.World.t3dFogRangeStart__
    del bpy.types.World.t3dFogRangeEnd__
    del bpy.types.World.t3dFogDithered__
    del bpy.types.World.t3dFogCurve__

    del bpy.types.Camera.t3dFOV__

    if exportOnSave in bpy.app.handlers.save_post:
        bpy.app.handlers.save_post.remove(exportOnSave)
    if onLoad in bpy.app.handlers.load_post:
        bpy.app.handlers.load_post.remove(onLoad)

if __name__ == "__main__":
    register()
