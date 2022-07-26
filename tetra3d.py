# Add-on for Tetra3D > Blender exporting

import bpy, os
from bpy.app.handlers import persistent

bl_info = {
    "name" : "Tetra3D Addon",                        # The name in the addon search menu
    "author" : "SolarLune Games",
    "description" : "An addon for exporting GLTF content from Blender for use with Tetra3D.",
    "blender" : (3, 0, 1),                             # Lowest version to use
    "location" : "View3D",
    "category" : "Gamedev",
    "version" : (0, 2),
    "support" : "COMMUNITY",
    "doc_url" : "https://github.com/SolarLune/Tetra3d/wiki/Blender-Addon",
}

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
    ("MULTIPLY", "Multiply", "Multiplicative fog - this fog mode darkens objects in the distance, with full effect being multiplying the object's color by the fog color at maximum distance (according to the camera's far range)", 0, 2),
    ("OVERWRITE", "Overwrite", "Overwrite fog - this fog mode overwrites the object's color with the fog color, with maximum distance being the camera's far distance", 0, 3),
]

gamePropTypes = [
    ("bool", "Bool", "Boolean data type", 0, 0),
    ("int", "Int", "Int data type", 0, 1),
    ("float", "Float", "Float data type", 0, 2),
    ("string", "String", "String data type", 0, 3),
    ("reference", "Object", "Object reference data type; converted to a string composed as follows on export - [SCENE NAME]:[OBJECT NAME]", 0, 4),
    ("color", "Color", "Color data type", 0, 5),
    ("vector3d", "3D Vector", "3D vector data type", 0, 6),
]

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
    # valueVector4D: bpy.props.FloatVectorProperty(name = "", description="The 4D vector value of the property")
    

class OBJECT_OT_tetra3dAddProp(bpy.types.Operator):
    bl_idname = "object.tetra3daddprop"
    bl_label = "Add Game Property"
    bl_description= "Adds a game property to the currently selected object. A game property gets added to an Object's Tags object in Tetra3D"
    bl_options = {'REGISTER', 'UNDO'}

    def execute(self, context):
        context.object.t3dGameProperties__.add()
        return {'FINISHED'}

class OBJECT_OT_tetra3dDeleteProp(bpy.types.Operator):
    bl_idname = "object.tetra3ddeleteprop"
    bl_label = "Delete Game Property"
    bl_description= "Deletes a game property from the currently selected object"
    bl_options = {'REGISTER', 'UNDO'}

    index : bpy.props.IntProperty()

    def execute(self, context):
        context.object.t3dGameProperties__.remove(self.index)
        return {'FINISHED'}

class OBJECT_OT_tetra3dReorderProps(bpy.types.Operator):
    bl_idname = "object.tetra3dreorderprops"
    bl_label = "Re-order Game Property"
    bl_description= "Moves a game property up or down in the list"
    bl_options = {'REGISTER', 'UNDO'}

    index : bpy.props.IntProperty()
    moveUp : bpy.props.BoolProperty()

    def execute(self, context):
        if self.moveUp:
            context.object.t3dGameProperties__.move(self.index, self.index-1)
        else:
            context.object.t3dGameProperties__.move(self.index, self.index+1)
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

   def execute(self, context):

        obj = context.object

        for o in context.selected_objects:
            o.t3dGameProperties__.clear()

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
        row.label(text="Node Path: " + objectNodePath(context.object))
        row = self.layout.row()
        row.operator("object.tetra3dcopynodepath", text="Copy Node Path to Clipboard", icon="COPYDOWN")
        
        row = self.layout.row()
        row.prop(context.object, "t3dVisible__")
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
        row = self.layout.row()
        row.separator()
        row = self.layout.row()
        row.operator("object.tetra3daddprop", text="Add Game Property", icon="PLUS")
        row.operator("object.tetra3dcopyprops", text="Overwrite All Game Properties", icon="COPYDOWN")

        for index, prop in enumerate(context.object.t3dGameProperties__):
            box = self.layout.box()
            row = box.row()
            row.prop(prop, "name")
            
            moveUpOptions = row.operator(OBJECT_OT_tetra3dReorderProps.bl_idname, text="", icon="TRIA_UP")
            moveUpOptions.index = index
            moveUpOptions.moveUp = True

            moveDownOptions = row.operator(OBJECT_OT_tetra3dReorderProps.bl_idname, text="", icon="TRIA_DOWN")
            moveDownOptions.index = index
            moveDownOptions.moveUp = False

            copy = row.operator(OBJECT_OT_tetra3dCopyOneProperty.bl_idname, text="", icon="COPYDOWN")
            copy.index = index

            deleteOptions = row.operator(OBJECT_OT_tetra3dDeleteProp.bl_idname, text="", icon="TRASH")
            deleteOptions.index = index

            row = box.row()
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
                row.prop(prop, "valueVector3D")
        
        row = self.layout.row()
        row.operator("object.tetra3dclearprops", text="Clear All Game Properties", icon="CANCEL")
        
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
        row.prop(context.material, "use_backface_culling")
        row = self.layout.row()
        row.prop(context.material, "blend_method")
        row = self.layout.row()
        row.prop(context.material, "t3dCompositeMode__")
        row = self.layout.row()
        row.prop(context.material, "t3dBillboardMode__")

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
        row.label(text="Fog Mode:")
        row.prop(context.world, "t3dFogMode__", text="Fog Mode:", expand=True)

        if context.world.t3dFogMode__ != "OFF":
            # box = self.layout.row()
            box.prop(context.world, "t3dFogColor__")
            
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


def export():
    scene = bpy.context.scene
        
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
        if "t3dFogColor__" in world:
            worldData["fog color"] = world.t3dFogColor__
        if "t3dFogRangeStart__" in world:
            worldData["fog range start"] = world.t3dFogRangeStart__
        if "t3dFogRangeEnd__" in world:
            worldData["fog range end"] = world.t3dFogRangeEnd__

        worlds[world.name] = worldData

    globalSet("t3dWorlds__", worlds)

    for scene in bpy.data.scenes:

        if scene.users > 0:

            if scene.world:
                scene["t3dCurrentWorld__"] = scene.world.name
            
            for layer in scene.view_layers:
                for obj in layer.objects:

                    obj["t3dOriginalLocalPosition__"] = obj.location

                    if obj.type == "MESH":
                        vertexColors = [layer.name for layer in obj.data.vertex_colors]
                        obj.data["t3dVertexColorNames__"] = vertexColors

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

    # Gather marker information and put them into the actions.
    for action in bpy.data.actions:
        markers = []
        for marker in action.pose_markers:
            markerInfo = {
                "name": marker.name,
                "time": marker.frame / scene.render.fps,
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

        export_extras=True,
        export_yup=True,
        export_apply=True,
    )

    # Undo changes that we've made after export

    for scene in bpy.data.scenes:

        if scene.world and "t3dCurrentWorld__" in scene:
            del(scene["t3dCurrentWorld__"])

        if scene.users > 0:

            for layer in scene.view_layers:

                for obj in layer.objects:

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
                    if obj.type == "MESH" and "t3dVertexColorNames__" in obj.data:
                        del(obj.data["t3dVertexColorNames__"])

    for action in bpy.data.actions:
        if "t3dMarkers__" in action:
            del(action["t3dMarkers__"])

    globalDel("t3dCollections__")
    globalDel("t3dWorlds__")

    return True

@persistent
def exportOnSave(dummy):
    
    if globalGet("t3dExportOnSave__"):
        export()


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
    "t3dCapsuleCustomEnabled__" : bpy.props.BoolProperty(name="Custom Capsule Size", description="If enabled, you can manually set the BoundingCapsule node's size properties. If disabled, the Capsule's size will be automatically determined by this object's mesh (if it is a mesh; otherwise, no BoundingCapsule node will be generated)", default=False),
    "t3dCapsuleCustomRadius__" : bpy.props.FloatProperty(name="Radius", description="The radius of the BoundingCapsule node.", min=0.0, default=0.5),
    "t3dCapsuleCustomHeight__" : bpy.props.FloatProperty(name="Height", description="The height of the BoundingCapsule node.", min=0.0, default=2),
    "t3dSphereCustomEnabled__" : bpy.props.BoolProperty(name="Custom Sphere Size", description="If enabled, you can manually set the BoundingSphere node's radius. If disabled, the Sphere's size will be automatically determined by this object's mesh (if it is a mesh; otherwise, no BoundingSphere node will be generated)", default=False),
    "t3dSphereCustomRadius__" : bpy.props.FloatProperty(name="Radius", description="Radius of the BoundingSphere node that will be created", min=0.0, default=1),
    "t3dGameProperties__" : bpy.props.CollectionProperty(type=t3dGamePropertyItem__)
}

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

def register():
    
    bpy.utils.register_class(OBJECT_PT_tetra3d)
    bpy.utils.register_class(RENDER_PT_tetra3d)
    bpy.utils.register_class(MATERIAL_PT_tetra3d)
    bpy.utils.register_class(WORLD_PT_tetra3d)
    bpy.utils.register_class(OBJECT_OT_tetra3dAddProp)
    bpy.utils.register_class(OBJECT_OT_tetra3dDeleteProp)
    bpy.utils.register_class(OBJECT_OT_tetra3dReorderProps)
    bpy.utils.register_class(OBJECT_OT_tetra3dCopyProps)
    bpy.utils.register_class(OBJECT_OT_tetra3dCopyOneProperty)
    bpy.utils.register_class(OBJECT_OT_tetra3dClearProps)
    bpy.utils.register_class(OBJECT_OT_tetra3dCopyNodePathToClipboard)
    bpy.utils.register_class(EXPORT_OT_tetra3d)
    
    bpy.utils.register_class(t3dGamePropertyItem__)

    for propName, prop in objectProps.items():
        setattr(bpy.types.Object, propName, prop)

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

    bpy.types.Material.t3dMaterialColor__ = bpy.props.FloatVectorProperty(name="Material Color", description="Material modulation color", default=[1,1,1,1], subtype="COLOR", size=4, step=1, min=0, max=1)
    bpy.types.Material.t3dMaterialShadeless__ = bpy.props.BoolProperty(name="Shadeless", description="Whether lighting should affect this material", default=False)
    bpy.types.Material.t3dCompositeMode__ = bpy.props.EnumProperty(items=materialCompositeModes, name="Composite Mode", description="Composite mode (i.e. additive, multiplicative, etc) for this material", default="DEFAULT")
    bpy.types.Material.t3dBillboardMode__ = bpy.props.EnumProperty(items=materialBillboardModes, name="Billboarding Mode", description="Billboard mode (i.e. if the object with this material should rotate to face the camera) for this material", default="NONE")
    
    bpy.types.World.t3dClearColor__ = bpy.props.FloatVectorProperty(name="Clear Color", description="Screen clear color; note that this won't actually be the background color automatically, but rather is simply set on the Scene.ClearColor property for you to use as you wish", default=[0.007, 0.008, 0.01, 1], subtype="COLOR", size=4, step=1, min=0, max=1)
    bpy.types.World.t3dFogColor__ = bpy.props.FloatVectorProperty(name="Fog Color", description="Fog color", default=[0, 0, 0, 1], subtype="COLOR", size=4, step=1, min=0, max=1)
    bpy.types.World.t3dFogMode__ = bpy.props.EnumProperty(items=worldFogCompositeModes, name="Fog Mode", description="Fog mode", default="OFF")

    bpy.types.World.t3dFogRangeStart__ = bpy.props.FloatProperty(name="Fog Range Start", description="With 0 being the near plane and 1 being the far plane of the camera, how far in should the fog start to appear", min=0.0, max=1.0, default=0, get=fogRangeStartGet, set=fogRangeStartSet)
    bpy.types.World.t3dFogRangeEnd__ = bpy.props.FloatProperty(name="Fog Range End", description="With 0 being the near plane and 1 being the far plane of the camera, how far out should the fog be at maximum opacity", min=0.0, max=1.0, default=1, get=fogRangeEndGet, set=fogRangeEndSet)

    if not exportOnSave in bpy.app.handlers.save_post:
        bpy.app.handlers.save_post.append(exportOnSave)
    
def unregister():
    bpy.utils.unregister_class(OBJECT_PT_tetra3d)
    bpy.utils.unregister_class(RENDER_PT_tetra3d)
    bpy.utils.unregister_class(MATERIAL_PT_tetra3d)
    bpy.utils.unregister_class(WORLD_PT_tetra3d)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dAddProp)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dDeleteProp)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dReorderProps)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dCopyProps)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dCopyOneProperty)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dClearProps)
    bpy.utils.unregister_class(OBJECT_OT_tetra3dCopyNodePathToClipboard)
    bpy.utils.unregister_class(EXPORT_OT_tetra3d)
    
    bpy.utils.unregister_class(t3dGamePropertyItem__)
    
    for propName, prop in objectProps.items():
        delattr(bpy.types.Object, propName)

    del bpy.types.Scene.t3dExportOnSave__
    del bpy.types.Scene.t3dExportFilepath__
    del bpy.types.Scene.t3dExportFormat__
    del bpy.types.Scene.t3dExportCameras__
    del bpy.types.Scene.t3dExportLights__
    del bpy.types.Scene.t3dPackTextures__

    del bpy.types.Material.t3dMaterialColor__
    del bpy.types.Material.t3dMaterialShadeless__
    del bpy.types.Material.t3dCompositeMode__
    del bpy.types.Material.t3dBillboardMode__

    del bpy.types.World.t3dClearColor__
    del bpy.types.World.t3dFogColor__
    del bpy.types.World.t3dFogMode__
    del bpy.types.World.t3dFogRangeStart__
    del bpy.types.World.t3dFogRangeEnd__

    if exportOnSave in bpy.app.handlers.save_post:
        bpy.app.handlers.save_post.remove(exportOnSave)
    

if __name__ == "__main__":
    register()
